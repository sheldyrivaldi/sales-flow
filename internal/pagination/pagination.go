// Package pagination normalizes page/page_size values so every list
// endpoint enforces the same bounds and echoes back the value it actually
// used to query.
package pagination

const (
	DefaultSize = 20
	MaxSize     = 100
)

// Normalize clamps page to >=1 and pageSize to [1, MaxSize], falling back to
// DefaultSize when pageSize is out of range.
func Normalize(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > MaxSize {
		pageSize = DefaultSize
	}
	return page, pageSize
}
