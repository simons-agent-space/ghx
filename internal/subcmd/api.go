package subcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/simons-agent-space/ghx/internal/api"
)

// API is the raw REST passthrough for anything the subcommands don't
// cover. METHOD is GET/POST/PATCH/PUT/DELETE. PATH is either a path
// (appended to the GitHub API base) or a full URL starting with
// https://api.github.com. -d BODY is a JSON-encoded value (object,
// array, number, bool, or null) or a bare string if it does not parse
// as JSON. To send a literal string that happens to look like JSON,
// wrap it in extra quotes: -d '"true"' sends the JSON string "true".
func API(ctx context.Context, c *api.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ghx api METHOD PATH [-d BODY]")
	}
	method := strings.ToUpper(args[0])
	path := args[1]
	if err := validateAPIPath(path); err != nil {
		return err
	}
	body := any(nil)
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "-d", "--data":
			if i+1 >= len(args) {
				return fmt.Errorf("-d requires a value")
			}
			i++
			// Allow JSON or bare string.
			// Heuristic: if the arg parses as JSON, use the parsed value;
			// otherwise treat it as a Go string. Note: -d null parses to
			// nil and results in no body sent; -d "" falls through to a
			// JSON-quoted empty string; -d 123 parses to a numeric body.
			var parsed any
			if err := json.Unmarshal([]byte(args[i]), &parsed); err == nil {
				body = parsed
			} else {
				body = args[i]
			}
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	var out any
	var err error
	switch method {
	case "GET":
		err = c.Get(ctx, path, &out)
	case "POST":
		err = c.Post(ctx, path, body, &out)
	case "PATCH":
		err = c.Patch(ctx, path, body, &out)
	case "PUT":
		err = c.Put(ctx, path, body, &out)
	case "DELETE":
		err = c.Delete(ctx, path, body, &out)
	default:
		return fmt.Errorf("unsupported method %q (use GET/POST/PATCH/PUT/DELETE)", method)
	}
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// validateAPIPath rejects http:// and any https:// URL whose host is
// not api.github.com. The client.go do() also enforces this as defense
// in depth (only attaches Authorization when the URL is under BaseURL).
func validateAPIPath(path string) error {
	if strings.HasPrefix(path, "http://") {
		return fmt.Errorf("http:// is not allowed; use https:// or a path")
	}
	if strings.HasPrefix(path, "https://") && !strings.HasPrefix(path, "https://api.github.com") {
		return fmt.Errorf("only https://api.github.com URLs allowed; got %q", path)
	}
	return nil
}
