package config

import _ "embed"

// Default is the built-in configuration shipped with the binary. It is used as
// the final fallback by Load when no user or project config file is found, and
// as the source written to disk by the `init` command. Embedding it guarantees
// gitreport works out of the box without requiring access to the source tree.
//
//go:embed default.yaml
var Default []byte
