// Package data embeds the bundled game data pack (see docs/SCOPE.md §4).
//
// Every entry carries a "verified" flag: values that have not been checked
// against a datamine or in-game measurement stay false and the UI must say so.
package data

import "embed"

//go:embed *.json
var FS embed.FS
