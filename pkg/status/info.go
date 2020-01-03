package status

import "time"

// Info holds the status info for an upload or download
type Info struct {
	Ref       string
	Status    string
	Offset    int64
	Total     int64
	StartedAt time.Time
	UpdatedAt time.Time
}

type Event struct {
	ID   string
	Name string
}
