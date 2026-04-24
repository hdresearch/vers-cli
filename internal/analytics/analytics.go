package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
)

const (
	eventCLICommandInvoked = "cli_command_invoked"
	eventCLICommandFailed  = "cli_command_failed"
)

// deniedPropertyKey mirrors the server-side denylist in
// vers-landing/src/app/api/telemetry/cli/route.ts. Keeping them in sync
// gives us defense-in-depth: filter client-side so sensitive-looking data
// never leaves the machine, and filter server-side in case a buggy client
// ships past this one.
var deniedPropertyKey = regexp.MustCompile(`(?i)(path|token|secret|key|ssh|env|command_line|argv|stdout|stderr)`)

// sanitizeProperties returns a copy of m with any keys matching the denylist
// removed. Returns nil for a nil input.
func sanitizeProperties(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if deniedPropertyKey.MatchString(k) {
			continue
		}
		out[k] = v
	}
	return out
}

type operation struct {
	Type       string         `json:"type"`
	Event      string         `json:"event,omitempty"`
	DistinctID string         `json:"distinct_id,omitempty"`
	Alias      string         `json:"alias,omitempty"`
	GroupType  string         `json:"group_type,omitempty"`
	GroupKey   string         `json:"group_key,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	// SetOnce is only meaningful for identify operations. Landing maps it to
	// PostHog's `$set_once` so immutable person properties (signed_up_at, etc.)
	// don't get overwritten on subsequent identify calls.
	SetOnce   map[string]any `json:"set_once,omitempty"`
	Timestamp string         `json:"timestamp,omitempty"`
}

type envelope struct {
	Source      string      `json:"source"`
	CLIVersion  string      `json:"cli_version"`
	GitCommit   string      `json:"git_commit,omitempty"`
	DeviceID    string      `json:"device_id"`
	SessionID   string      `json:"session_id"`
	AnonymousID string      `json:"anonymous_id,omitempty"`
	UserIDHint  string      `json:"user_id_hint,omitempty"`
	OrgIDHint   string      `json:"org_id_hint,omitempty"`
	Events      []operation `json:"events"`
}

type meResponse struct {
	UserID       string `json:"user_id"`
	DefaultOrgID string `json:"default_org_id"`
	Email        string `json:"email"`
}

type Client struct {
	enabled    bool
	debug      bool
	siteURL    string
	version    string
	gitCommit  string
	sessionID  string
	apiKey     string
	config     *auth.Config
	httpClient *http.Client
	queue      []operation
}

func New(version, gitCommit string, debug bool) (*Client, error) {
	config, err := auth.LoadConfig()
	if err != nil {
		return nil, err
	}
	enabled := telemetryEnabled(config)
	siteURL := ""
	if enabled {
		site, err := auth.GetSiteURL()
		if err != nil {
			return nil, err
		}
		siteURL = strings.TrimSuffix(site.String(), "/")
	}
	apiKey, _ := auth.GetAPIKey()

	client := &Client{
		enabled:   enabled,
		debug:     debug,
		siteURL:   siteURL,
		version:   version,
		gitCommit: gitCommit,
		sessionID: newUUID(),
		apiKey:    apiKey,
		config:    config,
		httpClient: &http.Client{
			// 500ms: landing-side `after()` + Node runtime + waitUntil means
			// events land even if we give up waiting for a response. This just
			// governs CLI exit speed for warm paths.
			Timeout: 500 * time.Millisecond,
		},
	}
	if !enabled {
		return client, nil
	}

	dirty := false
	if client.config.AnonymousID == "" {
		client.config.AnonymousID = newUUID()
		dirty = true
	}
	if client.config.DeviceID == "" {
		client.config.DeviceID = newUUID()
		dirty = true
	}
	if dirty {
		if err := auth.SaveConfig(client.config); err != nil && client.debug {
			fmt.Printf("[telemetry] failed to persist ids: %v\n", err)
		}
	}
	return client, nil
}

func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}

func (c *Client) SetAPIKey(apiKey string) {
	if c == nil {
		return
	}
	c.apiKey = apiKey
}

// BeginAuthenticationFlow detaches telemetry for a fresh login/signup flow from
// any previously cached authenticated session on this machine. The new flow
// gets a fresh anonymous ID so aliasing cannot accidentally merge a prior
// user's anonymous history into the next authenticated account.
func (c *Client) BeginAuthenticationFlow() {
	if c == nil || c.config == nil {
		return
	}
	c.apiKey = ""
	auth.ClearUserIdentity(c.config)
	c.config.AnonymousID = auth.NewTelemetryID()
	if c.config.DeviceID == "" {
		c.config.DeviceID = auth.NewTelemetryID()
	}
}

func (c *Client) AnonymousID() string {
	if c == nil || c.config == nil {
		return ""
	}
	return c.config.AnonymousID
}

func (c *Client) ReplaceConfig(config *auth.Config) {
	if c == nil || config == nil {
		return
	}
	c.config = config
}

func (c *Client) TrackCapture(event string, properties map[string]any) {
	if !c.Enabled() {
		return
	}
	c.queue = append(c.queue, operation{
		Type:       "capture",
		Event:      event,
		DistinctID: c.currentDistinctID(),
		Properties: sanitizeProperties(properties),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})
}

func (c *Client) TrackOutcome(event string, success bool, err error, properties map[string]any) {
	if !c.Enabled() {
		return
	}
	props := map[string]any{
		"success":     success,
		"error_class": nil,
	}
	for key, value := range properties {
		props[key] = value
	}
	if err != nil {
		props["error_class"] = classifyError(err)
	}
	c.TrackCapture(event, props)
}

func (c *Client) TrackCommandResult(command string, success bool, exitCode int, duration time.Duration, err error) {
	if !c.Enabled() || command == "" {
		return
	}
	properties := map[string]any{
		"command":       command,
		"success":       success,
		"exit_code":     exitCode,
		"duration_ms":   duration.Milliseconds(),
		"error_class":   nil,
		"authenticated": c.apiKey != "",
	}
	if err != nil {
		properties["error_class"] = classifyError(err)
	}
	c.TrackCapture(eventCLICommandInvoked, properties)
	if !success {
		c.TrackCapture(eventCLICommandFailed, properties)
	}
}

func (c *Client) TrackAuthSuccess(event, method, userID, orgID, email string) {
	if !c.Enabled() {
		return
	}
	updated := false
	if userID != "" && c.config.UserID != userID {
		c.config.UserID = userID
		updated = true
	}
	if orgID != "" && c.config.OrgID != orgID {
		c.config.OrgID = orgID
		updated = true
	}
	if email != "" && c.config.Email != email {
		c.config.Email = email
		updated = true
	}
	if updated {
		_ = auth.SaveConfig(c.config)
	}
	if c.apiKey != "" && c.config.AnonymousID != "" {
		c.queue = append(c.queue, operation{
			Type:       "alias",
			DistinctID: c.config.UserID,
			Alias:      c.config.AnonymousID,
		})
	}
	if c.apiKey != "" {
		// $set: mutable, overwritten on every identify. Only include fields
		// we actually have values for — NEVER send `email: null` on a
		// --token login where we don't know the email, because PostHog would
		// wipe the existing value set by an earlier web signup.
		setProps := map[string]any{
			"last_cli_version": c.version,
		}
		if c.config.Email != "" {
			setProps["email"] = c.config.Email
		}

		// $set_once: first write wins. first_cli_version is truly "first";
		// signed_up_at / entry_surface are signup-provenance metadata that
		// should never be overwritten by later CLI activity.
		setOnce := map[string]any{
			"first_cli_version": c.version,
		}
		if event == "signup_completed" {
			setOnce["signed_up_at"] = time.Now().UTC().Format(time.RFC3339)
			setOnce["entry_surface"] = "cli"
		}

		c.queue = append(c.queue, operation{
			Type:       "identify",
			DistinctID: c.config.UserID,
			Properties: sanitizeProperties(setProps),
			SetOnce:    sanitizeProperties(setOnce),
		})
	}
	if c.apiKey != "" {
		c.queue = append(c.queue, operation{
			Type:      "group_identify",
			GroupType: "organization",
			GroupKey:  c.config.OrgID,
			Properties: sanitizeProperties(map[string]any{
				"source": "vers-cli",
			}),
		})
	}
	c.TrackCapture(event, map[string]any{
		"source":      "vers-cli",
		"method":      method,
		"success":     true,
		"error_class": nil,
	})
}

func (c *Client) SyncIdentity() error {
	if !c.Enabled() || c.apiKey == "" {
		return nil
	}
	if c.config != nil && c.config.UserID != "" {
		return nil
	}
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		c.siteURL+"/api/telemetry/cli/me",
		nil,
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telemetry me returned %d", resp.StatusCode)
	}
	var me meResponse
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return err
	}
	changed := false
	if me.UserID != "" && c.config.UserID != me.UserID {
		c.config.UserID = me.UserID
		changed = true
	}
	if me.DefaultOrgID != "" && c.config.OrgID != me.DefaultOrgID {
		c.config.OrgID = me.DefaultOrgID
		changed = true
	}
	if me.Email != "" && c.config.Email != me.Email {
		c.config.Email = me.Email
		changed = true
	}
	if changed {
		return auth.SaveConfig(c.config)
	}
	return nil
}

func (c *Client) Flush() {
	if !c.Enabled() {
		return
	}
	c.flushQueue()
}

// flushQueue sends the queued events regardless of c.enabled. Callers that
// need to emit a single event even when telemetry is disabled (e.g. the
// telemetry_disabled toggle itself) may call this directly after ensuring
// DO_NOT_TRACK is honored.
func (c *Client) flushQueue() {
	if c == nil || len(c.queue) == 0 || c.siteURL == "" {
		return
	}
	body := envelope{
		Source:      "vers-cli",
		CLIVersion:  c.version,
		GitCommit:   c.gitCommit,
		DeviceID:    c.config.DeviceID,
		SessionID:   c.sessionID,
		AnonymousID: c.config.AnonymousID,
		UserIDHint:  c.currentUserIDHint(),
		OrgIDHint:   c.currentOrgIDHint(),
		Events:      c.queue,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.siteURL+"/api/telemetry/cli", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vers-CLI-Version", c.version)
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.debug {
			fmt.Printf("[telemetry] flush failed: %v\n", err)
		}
		return
	}
	defer resp.Body.Close()
	if c.debug {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("[telemetry] flush status=%d body=%s\n", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	c.queue = nil
}

// TrackToggle emits a single telemetry_enabled / telemetry_disabled event and
// flushes immediately. Bypasses the c.enabled gate so it fires even when the
// user has telemetry turned off in .versrc. The only absolute kill switch is
// DO_NOT_TRACK — that always wins.
func (c *Client) TrackToggle(newEnabled bool, success bool, err error) {
	if c == nil {
		return
	}
	if value := strings.ToLower(getenv("DO_NOT_TRACK")); value == "1" || value == "true" {
		return
	}
	if c.config == nil {
		cfg, err := auth.LoadConfig()
		if err != nil {
			return
		}
		c.config = cfg
	}
	if c.siteURL == "" {
		site, err := auth.GetSiteURL()
		if err != nil {
			return
		}
		c.siteURL = strings.TrimSuffix(site.String(), "/")
	}
	dirty := false
	if c.config.AnonymousID == "" {
		c.config.AnonymousID = auth.NewTelemetryID()
		dirty = true
	}
	if c.config.DeviceID == "" {
		c.config.DeviceID = auth.NewTelemetryID()
		dirty = true
	}
	if dirty {
		_ = auth.SaveConfig(c.config)
	}
	if c.apiKey == "" {
		if k, _ := auth.GetAPIKey(); k != "" {
			c.apiKey = k
		}
	}
	event := "telemetry_disabled"
	if newEnabled {
		event = "telemetry_enabled"
	}
	c.queue = append(c.queue, operation{
		Type:       "capture",
		Event:      event,
		DistinctID: c.currentDistinctID(),
		Properties: sanitizeProperties(map[string]any{
			"source":      "vers-cli",
			"enabled":     newEnabled,
			"success":     success,
			"error_class": errorClassOrNil(err),
		}),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	c.flushQueue()
}

func errorClassOrNil(err error) any {
	if err == nil {
		return nil
	}
	return classifyError(err)
}

func telemetryEnabled(config *auth.Config) bool {
	enabled, _ := auth.EffectiveTelemetryStatus(config)
	return enabled
}

func getenv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "auth"), strings.Contains(msg, "login"), strings.Contains(msg, "api key"), strings.Contains(msg, "unauthorized"), strings.Contains(msg, "authentication required"):
		return "auth_failed"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "connection"), strings.Contains(msg, "network"), strings.Contains(msg, "dial"), strings.Contains(msg, "lookup"), strings.Contains(msg, "request failed"):
		return "network"
	case strings.Contains(msg, "not found"):
		return "not_found"
	case strings.Contains(msg, "invalid"), strings.Contains(msg, "required"), strings.Contains(msg, "multiple organizations"), strings.Contains(msg, "organization"), strings.Contains(msg, "selection"):
		return "validation"
	default:
		return "internal"
	}
}

func (c *Client) currentDistinctID() string {
	if c != nil && c.apiKey != "" && c.config != nil && c.config.UserID != "" {
		return c.config.UserID
	}
	if c != nil && c.apiKey != "" {
		return ""
	}
	if c != nil && c.config != nil {
		return c.config.AnonymousID
	}
	return ""
}

func (c *Client) currentUserIDHint() string {
	if c != nil && c.apiKey != "" && c.config != nil {
		return c.config.UserID
	}
	return ""
}

func (c *Client) currentOrgIDHint() string {
	if c != nil && c.apiKey != "" && c.config != nil {
		return c.config.OrgID
	}
	return ""
}

func newUUID() string { return auth.NewTelemetryID() }
