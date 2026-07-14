// Package log configures structured (JSON) logging for the API (EP-17
// ST-17.2), replacing Echo's default plain-text request logger.
package log

import (
	"log/slog"
	"os"
)

// New returns a JSON slog.Logger writing to stdout — same destination
// Echo's previous text logger used, just machine-parseable instead of prose.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
