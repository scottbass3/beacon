package registry

import "time"

type Image struct {
	Name       string
	Repository string
	TagCount   int
	PullCount  int
	UpdatedAt  time.Time
}

type Project struct {
	Name       string
	ImageCount int
	UpdatedAt  time.Time
}

type Tag struct {
	Name         string
	Digest       string
	SizeBytes    int64
	UpdatedAt    time.Time
	PushedAt     time.Time
	LastPulledAt time.Time
}

type HistoryEntry struct {
	CreatedAt  time.Time
	CreatedBy  string
	Comment    string
	SizeBytes  int64
	EmptyLayer bool
}
