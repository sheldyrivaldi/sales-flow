package service

import (
	"context"
	"fmt"
	"strings"

	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
)

// negativeKeywordPreset is the deterministic Indonesia procurement negative-keyword
// catalog (ST-08.4 AC). It is always merged into the response so a keyword set is
// usable even when the AI call fails (degrade graceful).
var negativeKeywordPreset = []string{
	"hardware only",
	"pengadaan laptop",
	"sewa kendaraan",
	"ATK",
	"katering",
	"cleaning service",
	"jasa keamanan",
	"outsourcing tenaga kerja",
}

// keywordGenResult is the schema hint + unmarshal target passed to
// hermes.GenerateJSON.
type keywordGenResult struct {
	Keywords         []string `json:"keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
}

// KeywordService generates a draft keyword set from a set of service
// categories, using Hermes GenerateJSON. It never persists — persistence goes
// through ProfileService.Save (PUT /api/profile) so the versioning invariant
// (one is_current row, full history) is never bypassed.
type KeywordService struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewKeywordService(hc hermes.Client, sk hermes.SessionKey) *KeywordService {
	return &KeywordService{hc: hc, sk: sk}
}

// Generate builds a prompt from categories and asks Hermes for search
// keywords + negative keywords. On any AI failure (Hermes down, timeout,
// invalid JSON) it degrades gracefully: Degraded=true, Keywords empty,
// NegativeKeywords falls back to the preset alone — callers keep a usable
// (if AI-less) result instead of an error.
func (s *KeywordService) Generate(ctx context.Context, categories []string, language string) (*dto.KeywordGenerateResponse, error) {
	if language == "" {
		language = "id"
	}

	prompt := buildKeywordPrompt(categories, language)

	var result keywordGenResult
	if _, err := s.hc.GenerateJSON(ctx, prompt, &result, s.sk); err != nil {
		return &dto.KeywordGenerateResponse{
			Keywords:         []string{},
			NegativeKeywords: dedupCaseInsensitive(negativeKeywordPreset),
			Language:         language,
			Degraded:         true,
		}, nil
	}

	merged := dedupCaseInsensitive(append(append([]string{}, negativeKeywordPreset...), result.NegativeKeywords...))

	return &dto.KeywordGenerateResponse{
		Keywords:         dedupCaseInsensitive(result.Keywords),
		NegativeKeywords: merged,
		Language:         language,
		Degraded:         false,
	}, nil
}

func buildKeywordPrompt(categories []string, language string) string {
	return fmt.Sprintf(
		"Kamu membantu tim sales perusahaan mencari tender/pengadaan yang relevan. "+
			"Perusahaan ini menjual kapabilitas berikut: %s. "+
			"Hasilkan daftar kata kunci pencarian tender (bahasa: %s) yang relevan dengan kapabilitas tersebut, "+
			"serta daftar kata kunci negatif (tender yang HARUS dihindari karena tidak relevan, mis. pengadaan barang murni). "+
			"Balas HANYA JSON dengan schema persis: "+
			`{"keywords": ["..."], "negative_keywords": ["..."]}`+". Tanpa penjelasan, tanpa markdown, tanpa code fence.",
		strings.Join(categories, ", "),
		language,
	)
}

// dedupCaseInsensitive removes duplicate strings (case-insensitive, trimmed)
// while preserving the first-seen casing and order.
func dedupCaseInsensitive(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}
