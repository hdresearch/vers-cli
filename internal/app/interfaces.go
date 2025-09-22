package app

import (
	"io"
	"time"
)

// Clock provides time for testability.
type Clock interface{ Now() time.Time }

// RealClock implements Clock using time.Now.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// Env abstracts environment variable access for testability.
type Env interface{ Get(key string) string }

// OSEnv implements Env using os.Getenv (defined in app.go to avoid import cycle).

// Output bundles standard IO streams.
type Output struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}
