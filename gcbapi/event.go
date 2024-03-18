package gcbapi

import (
	"time"
)

// Event is an external shape for [event.Event].
type Event struct {
	ID                   string    `json:"id"`
	Type                 string    `json:"type"`
	GameID               string    `json:"gameId"`
	Year                 int64     `json:"year"`
	Group                string    `json:"group"`
	Title                string    `json:"title"`
	ShortDescription     string    `json:"shortDescription"`
	LongDescription      string    `json:"longDescription"`
	EventType            string    `json:"eventType"`
	GameSystem           string    `json:"gameSystem"`
	RulesEdition         string    `json:"rulesEdition"`
	MinPlayers           int64     `json:"minPlayers"`
	MaxPlayers           int64     `json:"maxPlayers"`
	AgeRequired          string    `json:"ageRequired"`
	ExperienceRequired   string    `json:"experienceRequired"`
	MaterialsProvided    string    `json:"materialsProvided"`
	StartDateTime        time.Time `json:"startDateTime"`
	Duration             float64   `json:"duration"`
	EndDateTime          time.Time `json:"endDateTime"`
	GMNames              string    `json:"gmNames"`
	Website              string    `json:"website"`
	Email                string    `json:"email"`
	Tournament           string    `json:"tournament"`
	RoundNumber          int64     `json:"roundNumber"`
	TotalRounds          int64     `json:"totalRounds"`
	MinimumPlayTime      float64   `json:"minimumPlayTime"`
	AttendeeRegistration string    `json:"attendeeRegistration"`
	Cost                 float64   `json:"cost"`
	Location             string    `json:"location"`
	RoomName             string    `json:"roomName"`
	TableNumber          string    `json:"tableNumber"`
	SpecialCategory      string    `json:"specialCategory"`
	TicketsAvailableTime int64     `json:"ticketsAvailable"`
	LastModified         time.Time `json:"lastModified"`
	AlsoRuns             time.Time `json:"alsoRuns"`
	Prize                string    `json:"prize"`
	RulesComplexity      string    `json:"rulesComplexity"`
	OriginalOrder        int64     `json:"originalOrder"`
}

// EventSearchResponse respects JSON:API specification for a JSON
// document response on the event search endpoint
type EventSearchResponse struct {
	Links Links   `json:"links"`
	Data  []Event `json:"data"`
	Meta  struct {
		Total int64 `json:"total"`
	} `json:"meta"`
	Error *Error `json:"error,omitempty"`
}

// Pagination implements the JSON:API [Pagination Object](https://jsonapi.org/format/#document-links)
type Pagination struct {
	First    string `json:"first,omitempty"`
	Last     string `json:"last,omitempty"`
	Previous string `json:"previous,omitempty"`
	Next     string `json:"next,omitempty"`
}

// Links implements the JSON:API [Links Object](https://jsonapi.org/format/#document-links)
type Links struct {
	Pagination
	Self string `json:"self"`
}

// Error implements the JSON:API [Error Object](https://jsonapi.org/format/#error-objects)
type Error struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}
