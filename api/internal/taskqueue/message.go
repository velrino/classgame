// Package taskqueue defines the provider-neutral message passed to job delivery
// adapters. Each workflow owns its enqueue port and JSON payload contract.
package taskqueue

import "time"

// Message describes delivery intent. Routing, transport authentication, retry
// policy and provider limits belong to the selected adapter and its deployment.
type Message struct {
	ID         string
	Payload    []byte
	Metadata   map[string]string
	ScheduleAt time.Time
}
