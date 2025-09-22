# Contributing to Vers CLI

This repo uses a thin-command architecture for the Cobra CLI. Command files under `cmd/` only parse flags/args and delegate to testable handlers in `internal/handlers/`. Handlers call domain services in `internal/services/` and results are rendered by `internal/presenters/`.

## Adding a New Command

1. Create a handler `internal/handlers/hello.go`:
```
package handlers

import (
  "context"
  "github.com/hdresearch/vers-cli/internal/app"
)

type HelloReq struct { Name string }
type HelloView struct { Greeting string }

func HandleHello(ctx context.Context, a *app.App, r HelloReq) (HelloView, error) {
  g := "Hello"
  if r.Name != "" { g += ", " + r.Name }
  return HelloView{ Greeting: g + "!" }, nil
}
```

2. Add a presenter `internal/presenters/hello_presenter.go`:
```
package presenters

import (
  "fmt"
  "github.com/hdresearch/vers-cli/internal/app"
)

func RenderHello(a *app.App, v interface{ Greeting string }) { fmt.Println(v.Greeting) }
```

3. Add a thin Cobra command in `cmd/hello.go`:
```
package cmd

import (
  "context"
  "github.com/hdresearch/vers-cli/internal/handlers"
  pres "github.com/hdresearch/vers-cli/internal/presenters"
  "github.com/spf13/cobra"
)

var helloCmd = &cobra.Command{
  Use: "hello [name]",
  RunE: func(cmd *cobra.Command, args []string) error {
    var name string
    if len(args) > 0 { name = args[0] }
    ctx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIShort)
    defer cancel()
    view, err := handlers.HandleHello(ctx, application, handlers.HelloReq{Name: name})
    if err != nil { return err }
    pres.RenderHello(application, view)
    return nil
  },
}

func init() { rootCmd.AddCommand(helloCmd) }
```

## SDK Requests

When the SDK expects `param.Field[T]` values, wrap scalars with `vers.F(value)`. See existing handlers (e.g., `run.go`, `run_commit.go`, `rename.go`).

## App Container

`cmd/root.go` creates a shared `application *app.App` with:
- Vers SDK client
- IO (In/Out/Err)
- Prompter (for confirmations)
- Exec runner (for ssh/scp)
- Timeouts (APIShort/APIMedium/APILong/BuildUpload/SSHConnect)

Handlers should use `application` for cross-cutting concerns (prompts, runner, timeouts), and services for backend operations.

## Tests

- Use unit tests for presenters and small utilities.
- Integration tests shell out to the CLI under `test/` (requires `.env` with `VERS_URL` (include scheme) and `VERS_API_KEY`).

