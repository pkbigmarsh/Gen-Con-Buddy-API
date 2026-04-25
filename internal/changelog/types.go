package changelog

import (
	"time"

	"github.com/google/uuid"
)

// Entry - Change Log Entry includes a reference to all of the modified events,
// and what modification action was performed on them.
type Entry struct {
	ID            string   `json:"id"`
	Date          string   `json:"date"`
	UpdatedEvents []string `json:"updatedEvents"`
	DeletedEvents []string `json:"deletedEvents"`
	CreatedEvents []string `json:"createdEvents"`
}

// NewEntry instantiates a [Entry] with a UUID for the ID
// and a date timestamp of now.
func NewEntry() *Entry {
	return &Entry{
		ID:            uuid.Must(uuid.NewV7()).String(),
		Date:          time.Now().Format(time.RFC3339),
		UpdatedEvents: []string{},
		DeletedEvents: []string{},
		CreatedEvents: []string{},
	}
}

// ListEntriesRequest fetches a specific number of entries
// sort by date in ascending order.
type ListEntriesRequest struct {
	Limit int
}

// FetchEntriesResponse contains maps for the found and missing [Entry]
type FetchEntriesResponse struct {
	Found   map[string]*Entry
	Missing map[string]struct{}
}
