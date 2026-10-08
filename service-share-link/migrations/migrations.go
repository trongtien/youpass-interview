// Package migrations embeds the SQL migrations so the binary can apply them
// without the files being shipped alongside it.
//
// Files follow golang-migrate naming: {version}_{name}.up.sql / .down.sql.
// Create a new pair with `make migrate-create name=<name>`.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
