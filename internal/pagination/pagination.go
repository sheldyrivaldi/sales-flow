// Package pagination normalizes page/page_size values so every list
// endpoint enforces the same bounds and echoes back the value it actually
// used to query.
package pagination

const (
	DefaultSize = 20
	// MaxSize must cover the largest "load everything in one page" view
	// (the prospect Kanban board fetches the whole pipeline at once).
	MaxSize = 500
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
