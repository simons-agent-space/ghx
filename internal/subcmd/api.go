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
// cover. METHOD is GET/POST/PATCH/PUT/DELETE. PATH is either a full
// URL or a path that will be appended to the GitHub API base. -d BODY
// is a JSON-encoded value (object or array) or a bare string.
func API(ctx context.Context, c *api.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ghx api METHOD PATH [-d BODY]")
	}
	method := strings.ToUpper(args[0])
	path := args[1]
	body := any(nil)
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "-d", "--data":
			if i+1 >= len(args) {
				return fmt.Errorf("-d requires a value")
			}
			i++
			// Allow JSON or bare string.
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
		err = c.Delete(ctx, path, &out)
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
