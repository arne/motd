// Package motd embeds shipped assets (animal marks, the example config) so
// the binary can render a useful default when no config file is present.
package motd

import "embed"

//go:embed examples/marks/animals/*.ansi
var Marks embed.FS

//go:embed examples/config.yaml
var ExampleConfig string
