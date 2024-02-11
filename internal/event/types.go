package event

import "time"

type Event struct {
	GameID               string       `json:"game_id"`
	Year                 int          `json:"year"`
	Group                string       `json:"group"`
	Title                string       `json:"title"`
	ShortDescription     string       `json:"short_description"`
	LongDescription      string       `json:"long_description"`
	EventType            Type         `json:"event_type"`
	GameSystem           string       `json:"game_system"`
	RulesEdition         string       `json:"rules_edition"`
	MinPlayers           int          `json:"min_players"`
	MaxPlayers           int          `json:"max_players"`
	AgeRequired          AgeGroup     `json:"age_required"`
	ExperienceRequired   EXP          `json:"experience_required"`
	MaterialsProvided    string       `json:"materials_provided"`
	StartDateTime        time.Time    `json:"start_date_time"`
	Duration             float64      `json:"duration"`
	EndDateTime          time.Time    `json:"end_date_time"`
	GMNames              string       `json:"gm_names"`
	Website              string       `json:"website"`
	Email                string       `json:"email"`
	Tournament           string       `json:"tournament"`
	RoundNumber          int          `json:"round_number"`
	TotalRounds          int          `json:"total_rounds"`
	MinimumPlayTime      float64      `json:"minimum_play_time"`
	AttendeeRegistration Registration `json:"attendee_registration"`
	Cost                 float64      `json:"cost"`
	Location             string       `json:"location"`
	RoomName             string       `json:"room_name"`
	TableNumber          string       `json:"table_number"`
	SpecialCategory      Category     `json:"special_category"`
	TicketsAvailableTime int          `json:"tickets_available"`
	LastModified         time.Time    `json:"last_modified"`
	AlsoRuns             time.Time    `json:"also_runs"`
	Prize                string       `json:"prize"`
	RulesComplexity      string       `json:"rules_complexity"`
}
