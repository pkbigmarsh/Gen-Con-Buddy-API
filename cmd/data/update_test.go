package data

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gencon_buddy_api/internal/changelog"
)

// makeChangeLogIDs returns n distinct placeholder event IDs.
func makeChangeLogIDs(prefix string, n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("%s%d", prefix, i)
	}

	return ids
}

func TestTrimChangeLogEntry(t *testing.T) {
	tests := []struct {
		name        string
		limit       int
		created     int
		deleted     int
		updated     int
		wantCreated int
		wantDeleted int
		wantUpdated int
	}{
		{
			name:        "under limit keeps everything",
			limit:       10,
			created:     2,
			deleted:     3,
			updated:     1,
			wantCreated: 2,
			wantDeleted: 3,
			wantUpdated: 1,
		},
		{
			name:        "exactly at limit keeps everything",
			limit:       10,
			created:     4,
			deleted:     3,
			updated:     3,
			wantCreated: 4,
			wantDeleted: 3,
			wantUpdated: 3,
		},
		{
			name:        "created alone exceeds limit drops deleted and updated",
			limit:       10,
			created:     15,
			deleted:     5,
			updated:     5,
			wantCreated: 10,
			wantDeleted: 0,
			wantUpdated: 0,
		},
		{
			name:        "created exactly at limit drops deleted and updated",
			limit:       10,
			created:     10,
			deleted:     4,
			updated:     4,
			wantCreated: 10,
			wantDeleted: 0,
			wantUpdated: 0,
		},
		{
			name:        "created plus deleted exceed limit drops updated and trims deleted",
			limit:       10,
			created:     4,
			deleted:     12,
			updated:     5,
			wantCreated: 4,
			wantDeleted: 6,
			wantUpdated: 0,
		},
		{
			name:        "created plus deleted exactly at limit drops updated keeps deleted",
			limit:       10,
			created:     4,
			deleted:     6,
			updated:     5,
			wantCreated: 4,
			wantDeleted: 6,
			wantUpdated: 0,
		},
		{
			name:        "updated overflow is trimmed keeping created and deleted",
			limit:       10,
			created:     3,
			deleted:     3,
			updated:     10,
			wantCreated: 3,
			wantDeleted: 3,
			wantUpdated: 4,
		},
		{
			name:        "empty entry is a no-op",
			limit:       10,
			created:     0,
			deleted:     0,
			updated:     0,
			wantCreated: 0,
			wantDeleted: 0,
			wantUpdated: 0,
		},
		{
			name:        "deletions dominate at the real 10k limit",
			limit:       changeLogEventLimit,
			created:     4000,
			deleted:     9000,
			updated:     2000,
			wantCreated: 4000,
			wantDeleted: 6000,
			wantUpdated: 0,
		},
		{
			name:        "created overflow at the real 10k limit",
			limit:       changeLogEventLimit,
			created:     25000,
			deleted:     8000,
			updated:     5000,
			wantCreated: changeLogEventLimit,
			wantDeleted: 0,
			wantUpdated: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := changelog.NewEntry()
			entry.CreatedEvents = makeChangeLogIDs("c", tt.created)
			entry.DeletedEvents = makeChangeLogIDs("d", tt.deleted)
			entry.UpdatedEvents = makeChangeLogIDs("u", tt.updated)

			trimChangeLogEntry(entry, tt.limit)

			require.Len(t, entry.CreatedEvents, tt.wantCreated, "CreatedEvents length")
			require.Len(t, entry.DeletedEvents, tt.wantDeleted, "DeletedEvents length")
			require.Len(t, entry.UpdatedEvents, tt.wantUpdated, "UpdatedEvents length")

			// The core data restriction: the recorded ID count never exceeds the limit.
			total := len(entry.CreatedEvents) + len(entry.DeletedEvents) + len(entry.UpdatedEvents)
			require.LessOrEqual(t, total, tt.limit, "total recorded IDs must not exceed the limit")
		})
	}
}
