package app

import "time"

// Timeouts centralize operation time budgets.
type Timeouts struct {
	APIShort    time.Duration
	APIMedium   time.Duration
	APILong     time.Duration
	BuildUpload time.Duration
	SSHConnect  time.Duration
}

// DefaultTimeouts returns conservative defaults aligning with current code.
func DefaultTimeouts() Timeouts {
	return Timeouts{
		APIShort:    10 * time.Second,
		APIMedium:   30 * time.Second,
		APILong:     60 * time.Second,
		BuildUpload: 600 * time.Second,
		SSHConnect:  5 * time.Second,
	}
}
