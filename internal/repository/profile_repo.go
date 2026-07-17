package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"salespilot/internal/domain"
)

var _ domain.ProfileRepository = (*ProfileRepo)(nil)

type ProfileRepo struct {
	db *gorm.DB
}

func NewProfileRepo(db *gorm.DB) *ProfileRepo {
	return &ProfileRepo{db: db}
}

// GetCurrent loads the aggregate whose CompanyProfile.IsCurrent=true, along
// with its child rows. Returns gorm.ErrRecordNotFound if no profile exists yet.
func (r *ProfileRepo) GetCurrent(ctx context.Context) (*domain.ProfileAggregate, error) {
	return loadCurrent(r.db.WithContext(ctx))
}

func loadCurrent(tx *gorm.DB) (*domain.ProfileAggregate, error) {
	var cp domain.CompanyProfile
	if err := tx.First(&cp, "is_current = true").Error; err != nil {
		return nil, fmt.Errorf("profile.GetCurrent: %w", err)
	}

	agg := &domain.ProfileAggregate{Profile: cp}

	var target domain.TargetCriteria
	if err := tx.First(&target, "profile_id = ?", cp.ID).Error; err == nil {
		agg.Target = &target
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("profile.GetCurrent target: %w", err)
	}

	var nogo domain.NoGoRule
	if err := tx.First(&nogo, "profile_id = ?", cp.ID).Error; err == nil {
		agg.NoGo = &nogo
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("profile.GetCurrent nogo: %w", err)
	}

	var keywords []domain.KeywordSet
	if err := tx.Where("profile_id = ?", cp.ID).Find(&keywords).Error; err != nil {
		return nil, fmt.Errorf("profile.GetCurrent keywords: %w", err)
	}
	agg.Keywords = keywords

	var scoring domain.ScoringConfig
	if err := tx.First(&scoring, "profile_id = ?", cp.ID).Error; err == nil {
		agg.ScoringConfig = &scoring
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("profile.GetCurrent scoring_config: %w", err)
	}

	return agg, nil
}

// CreateVersion clones agg into a brand-new CompanyProfile version: it flips
// off the prior IsCurrent row (if any) before inserting the new one, so the
// partial unique index on is_current never sees two true rows at once, then
// inserts child rows re-pointed at the new profile ID. The prior version's
// rows are left untouched (history).
func (r *ProfileRepo) CreateVersion(ctx context.Context, agg *domain.ProfileAggregate) (*domain.ProfileAggregate, error) {
	var result *domain.ProfileAggregate

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		nextVersion := 1
		var prev domain.CompanyProfile
		// Lock the current row FOR UPDATE so two concurrent CreateVersion calls
		// serialize: the second blocks here until the first commits, then re-reads
		// the freshly-inserted current row and derives the next version from it.
		// Without the lock both could read the same version N and both insert N+1.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&prev, "is_current = true").Error; err == nil {
			nextVersion = prev.Version + 1
			if err := tx.Model(&domain.CompanyProfile{}).
				Where("is_current = true").
				Update("is_current", false).Error; err != nil {
				return fmt.Errorf("profile.CreateVersion demote: %w", err)
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("profile.CreateVersion lookup: %w", err)
		}

		cp := agg.Profile
		cp.ID = ""
		cp.Version = nextVersion
		cp.IsCurrent = true
		if err := tx.Create(&cp).Error; err != nil {
			return fmt.Errorf("profile.CreateVersion insert profile: %w", err)
		}

		if agg.Target != nil {
			t := *agg.Target
			t.ID = ""
			t.ProfileID = cp.ID
			if err := tx.Create(&t).Error; err != nil {
				return fmt.Errorf("profile.CreateVersion insert target_criteria: %w", err)
			}
		}

		if agg.NoGo != nil {
			n := *agg.NoGo
			n.ID = ""
			n.ProfileID = cp.ID
			if err := tx.Create(&n).Error; err != nil {
				return fmt.Errorf("profile.CreateVersion insert nogo_rule: %w", err)
			}
		}

		for _, k := range agg.Keywords {
			k.ID = ""
			k.ProfileID = cp.ID
			if err := tx.Create(&k).Error; err != nil {
				return fmt.Errorf("profile.CreateVersion insert keyword_set: %w", err)
			}
		}

		if agg.ScoringConfig != nil {
			sc := *agg.ScoringConfig
			sc.ID = ""
			sc.ProfileID = cp.ID
			if err := tx.Create(&sc).Error; err != nil {
				return fmt.Errorf("profile.CreateVersion insert scoring_config: %w", err)
			}
		}

		loaded, err := loadCurrent(tx)
		if err != nil {
			return fmt.Errorf("profile.CreateVersion reload: %w", err)
		}
		result = loaded
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
