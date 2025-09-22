package app

import (
	"io"
	"net/url"
	"os"

	"github.com/hdresearch/vers-cli/internal/prompts"
	runrt "github.com/hdresearch/vers-cli/internal/runtime"
	vers "github.com/hdresearch/vers-sdk-go"
)

// App is the dependency container for command handlers.
type App struct {
	Client   *vers.Client
	IO       Output
	Prompter prompts.Prompter
	Runner   runrt.Runner
	Clock    Clock
	Env      Env
	Timeouts Timeouts
	BaseURL  *url.URL
	Verbose  bool
}

// New constructs an App.
func New(
	client *vers.Client,
	in io.Reader, out io.Writer, err io.Writer,
	prompter prompts.Prompter,
	runner runrt.Runner,
	baseURL *url.URL,
	verbose bool,
	timeouts Timeouts,
	env Env,
	clock Clock,
) *App {
	return &App{
		Client:   client,
		IO:       Output{In: in, Out: out, Err: err},
		Prompter: prompter,
		Runner:   runner,
		Clock:    clock,
		Env:      env,
		Timeouts: timeouts,
		BaseURL:  baseURL,
		Verbose:  verbose,
	}
}

// OSEnv implements Env via os.Getenv.
type OSEnv struct{}

func (OSEnv) Get(key string) string { return os.Getenv(key) }
