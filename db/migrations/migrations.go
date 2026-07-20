// Package migrations embeds the SQL migration files into the compiled Go
// binary via go:embed, so the API can run its own migrations on boot
// (see internal/repository/migrate.go) without depending on a separate
// `migrate` CLI, an init container, or docker-compose building a local
// postgres — required for environments (e.g. company Kubernetes cluster,
// external/company-managed database) where only the API image's own
// startup behavior can be relied upon.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
