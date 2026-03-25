package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	envsvc "github.com/hdresearch/vers-cli/internal/services/env"
)

// EnvListReq captures the request for listing env vars
type EnvListReq struct{}

// EnvSetReq captures the request for setting an env var
type EnvSetReq struct {
	Key   string
	Value string
}

// EnvDeleteReq captures the request for deleting an env var
type EnvDeleteReq struct {
	Key string
}

// HandleEnvList handles listing environment variables
func HandleEnvList(ctx context.Context, a *app.App, req EnvListReq) (map[string]string, error) {
	vars, err := envsvc.ListEnvVars(ctx, a.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to list environment variables: %w", err)
	}
	return vars, nil
}

// HandleEnvSet handles setting an environment variable
func HandleEnvSet(ctx context.Context, a *app.App, req EnvSetReq) error {
	if req.Key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	err := envsvc.SetEnvVar(ctx, a.Client, req.Key, req.Value)
	if err != nil {
		return fmt.Errorf("failed to set environment variable: %w", err)
	}
	return nil
}

// HandleEnvDelete handles deleting an environment variable
func HandleEnvDelete(ctx context.Context, a *app.App, req EnvDeleteReq) error {
	if req.Key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	err := envsvc.DeleteEnvVar(ctx, a.Client, req.Key)
	if err != nil {
		return fmt.Errorf("failed to delete environment variable: %w", err)
	}
	return nil
}
