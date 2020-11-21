package github

import (
	"context"

	"github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
)

// Syncer represents components whose responsibility is to keep Github resources
// (e.g. Webhooks, deploy keys, etc.) in sync with workflow's settings.
type Syncer interface {
	Sync(ctx context.Context, workflow *v1alpha1.Workflow) (*SyncResult, error)
}

// SyncResult stores information about a Github resource that has been synced.
type SyncResult struct {
	Entries []SyncResultEntry
}

// SyncResultEntry represents the result of the syncing process for a given resource.
type SyncResultEntry struct {
	ID         int64
	Repository *v1alpha1.Repository
	Action     ActionType
	Secret     []byte
}

// ActionType represents actions that may be taken by Github syncers to keep
// resources in sync.
type ActionType string

const (
	Created ActionType = "Created"
	Updated ActionType = "Updated"
	Deleted ActionType = "Deleted"
)

// EmptySyncResult returns a new SyncResult object with no entries.
func EmptySyncResult() *SyncResult {
	return &SyncResult{}
}

// Add adds a new SyncResultEntry to the SyncResult object in question.
func (s *SyncResult) Add(entry SyncResultEntry) *SyncResult {
	if s.Entries == nil {
		s.Entries = make([]SyncResultEntry, 0)
	}
	s.Entries = append(s.Entries, entry)
	return s
}

// HasCreatedResources returns true if new Github resources (e.g. Webhooks,
// deploy keys, etc.) were created during the sync process or false otherwise.
func (s *SyncResult) HasCreatedResources() bool {
	for _, entry := range s.Entries {
		if entry.Action == Created {
			return true
		}
	}
	return false
}
