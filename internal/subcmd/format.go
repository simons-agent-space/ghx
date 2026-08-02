// Package subcmd contains the ghx subcommand implementations.
//
// Each subcommand lives in its own file (pr_view.go, pr_list.go, etc.)
// and shares the helpers in this file. Subcommands receive an
// *api.Client and a parsed flag set, and print to os.Stdout.
package subcmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
)

// flagSet creates a flag set that exits cleanly on parse error and
// prints a one-line usage hint on misuse. The full usage text is
// printed by main when an unknown subcommand is given.
func flagSet(name, usageSuffix string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // suppress default usage on parse error
	fs.Usage = func() {
		fmt.Fprintf(io.Discard, "usage: ghx %s %s\n", name, usageSuffix)
	}
	return fs
}

// writeJSON writes v as indented JSON to w.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func boolWord(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
