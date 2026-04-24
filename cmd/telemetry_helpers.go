package cmd

import "time"

func trackSemanticEvent(event string, properties map[string]any) {
	if telemetryClient == nil {
		return
	}
	telemetryClient.TrackOutcome(event, true, nil, properties)
}

func trackSemanticOutcome(event string, err error, properties map[string]any) {
	if telemetryClient == nil {
		return
	}
	telemetryClient.TrackOutcome(event, err == nil, err, properties)
}

func trackSemanticOutcomeWithDuration(event string, err error, started time.Time, properties map[string]any) {
	if properties == nil {
		properties = map[string]any{}
	}
	properties["duration_ms"] = time.Since(started).Milliseconds()
	trackSemanticOutcome(event, err, properties)
}

func trackAuthFailure(event string, flow string, method string, err error) {
	trackSemanticOutcome(event, err, map[string]any{
		"flow":   flow,
		"method": method,
	})
}
