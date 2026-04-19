package gcbapi

// ChangeLogSummary is an external shape summarrizing how many events were changed.
type ChangeLogSummary struct {
	ID           string `json:"id"`
	Date         string `json:"date"`
	UpdatedCount int    `json:"updatedCount"`
	DeletedCount int    `json:"deletedCount"`
	CreatedCount int    `json:"createdCount"`
}

// ListChangeLogsResponse lists [ChangeLogSummay]s.
type ListChangeLogsResponse struct {
	Error   string             `json:"error,omitempty"`
	Entries []ChangeLogSummary `json:"entries,omitempty"`
}

// ChangeLogEntry includes fully hydrated events.
type ChangeLogEntry struct {
	ID            string  `json:"id"`
	Date          string  `json:"date"`
	UpdatedEvents []Event `json:"updatedEvents"`
	DeletedEvents []Event `json:"deletedEvents"`
	CreatedEvents []Event `json:"createdEvents"`
}

// FetchChangeLogResponse is the api response for the fetch actionz.
type FetchChangeLogResponse struct {
	Error string         `json:"error,omitempty"`
	Entry ChangeLogEntry `json:"entry,omitempty"`
}
