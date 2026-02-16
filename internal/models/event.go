package models

// EventIngestRequest is the POST /events payload.
// event_id is optional; best practice is to pass Idempotency-Key header for retries.
type EventIngestRequest struct {
	EventID    string                 `json:"event_id,omitempty"`
	EventName  string                 `json:"event_name"`
	Timestamp  string                 `json:"timestamp"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// EventIngestResponse is returned by POST /events.
// Duplicate indicates idempotent success (the event already existed).
type EventIngestResponse struct {
	EventID   string `json:"event_id"`
	Duplicate bool   `json:"duplicate"`
}
