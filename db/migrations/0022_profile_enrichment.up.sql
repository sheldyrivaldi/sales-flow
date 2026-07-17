-- Company profile: portfolio/evidence references (RFI §4.1 "Bukti/Portfolio").
ALTER TABLE company_profile ADD COLUMN portfolio_refs JSONB NOT NULL DEFAULT '[]';

-- Target criteria: buyer-profile fields with no home yet (RFI §5.1).
ALTER TABLE target_criteria ADD COLUMN buyer_size_note TEXT;
ALTER TABLE target_criteria ADD COLUMN document_languages JSONB NOT NULL DEFAULT '[]';
ALTER TABLE target_criteria ADD COLUMN work_model TEXT;
ALTER TABLE target_criteria ADD COLUMN onsite_limit_note TEXT;
ALTER TABLE target_criteria ADD COLUMN decision_maker_roles JSONB NOT NULL DEFAULT '[]';

-- Source: per-source monitoring cadence + data types (RFI §6.1).
ALTER TABLE source ADD COLUMN frequency TEXT NOT NULL DEFAULT 'harian'
    CHECK (frequency IN ('harian','2-3x','mingguan','manual'));
ALTER TABLE source ADD COLUMN data_types JSONB NOT NULL DEFAULT '[]';

-- Scoring config: configurable rubric weights + recommendation thresholds
-- (RFI §8), one row per profile version — same versioned-clone pattern as
-- target_criteria/nogo_rule. Defaults mirror today's hardcoded values in
-- internal/ai/scoring.go/recommend.go so an unconfigured workspace behaves
-- identically to before this migration.
CREATE TABLE scoring_config (
    id                                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id                          UUID          NOT NULL REFERENCES company_profile(id) ON DELETE CASCADE,
    weight_capability_fit               INT           NOT NULL DEFAULT 20,
    weight_portfolio_match              INT           NOT NULL DEFAULT 15,
    weight_commercial_attractiveness    INT           NOT NULL DEFAULT 15,
    weight_eligibility_fit              INT           NOT NULL DEFAULT 15,
    weight_deadline_feasibility         INT           NOT NULL DEFAULT 10,
    weight_strategic_account_value      INT           NOT NULL DEFAULT 10,
    weight_delivery_risk                INT           NOT NULL DEFAULT 10,
    weight_competition_win_probability  INT           NOT NULL DEFAULT 5,
    threshold_pursue                    INT           NOT NULL DEFAULT 80,
    threshold_review                    INT           NOT NULL DEFAULT 65,
    threshold_watchlist                 INT           NOT NULL DEFAULT 50,
    created_at                          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at                          TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ON scoring_config (profile_id);
