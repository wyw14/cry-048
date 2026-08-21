package domain

import (
	"strings"
	"time"
)

type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) Valid() bool {
	value := strings.TrimSpace(string(id))
	return value != "" && !strings.ContainsAny(value, "\\/\x00")
}

type Revision int64

func (revision Revision) Next() Revision {
	return revision + 1
}

func (revision Revision) Matches(expected Revision) bool {
	return revision == expected
}

type Actor struct {
	ID    ID
	Email string
	Name  string
}

func (actor Actor) Valid() bool {
	return actor.ID.Valid() && strings.Contains(actor.Email, "@")
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type FixedClock struct {
	Time time.Time
}

func (clock FixedClock) Now() time.Time {
	return clock.Time
}
