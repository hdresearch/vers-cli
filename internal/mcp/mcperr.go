package mcp

import "fmt"

// Simple error coding for MCP responses. The SDK accepts an error; we embed a code in the message
// and attach optional details for debugging.

const (
	E_INVALID          = "E_INVALID"
	E_CONFIRM_REQUIRED = "E_CONFIRM_REQUIRED"
	E_NOT_FOUND        = "E_NOT_FOUND"
	E_CONFLICT         = "E_CONFLICT"
	E_INTERNAL         = "E_INTERNAL"
)

type Error struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *Error) Error() string { return fmt.Sprintf("[%s] %s", e.Code, e.Message) }

func Err(code, msg string, details map[string]any) error {
	return &Error{Code: code, Message: msg, Details: details}
}
