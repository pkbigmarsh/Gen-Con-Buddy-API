package event

import (
	"fmt"

	"github.com/gencon_buddy_api/internal/search"
)

// SearchRequest contains all the needed information for searching
// for events
type SearchRequest struct {
	Terms []search.Term
	Page  int
	Limit int
}

type SearchResponse struct {
	TotalEvents int64
	Events      []*Event
}

func NewSearchField(f string, value string) (search.Term, error) {

	field, err := FieldFromString(f)
	if err != nil {
		return nil, err
	}

	switch field {
	case GameID, EventType, AgeRequired, ExperienceRequired,
		AttendeeRegistration, SpecialCategory:
		return search.NewKeyword(f, value)
	// integer
	case Year, MinPlayers, MaxPlayers, RoundNumber,
		TotalRounds, TicketsAvailable:
		return search.NewNumber(f, value)
	case Group, Title, ShortDescription, LongDescription, GameSystem,
		RulesEdition, MaterialsProvided, GMNames, Website, Email, Tournament,
		Location, RoomName, TableNumber, Prize, RulesComplexity:
		return search.NewText(f, value)
	// double
	case Duration, MinimumPlayTime, Cost:
		return search.NewNumber(f, value)
	case StartDateTime, EndDateTime, LastModified, AlsoRuns:
		return search.NewDate(f, value)
	case Filter:
		return FilterTerm{value: value}, nil
	default:
		return nil, fmt.Errorf("Field %s is not supported as a search field", f)
	}
}

// FilterTerm is a special [search.Term] implementation for a virtual "filter" field.
// This search matches against title, short description, and long description.
type FilterTerm struct {
	value string
}

func (f FilterTerm) ToQuery() (any, error) {
	return map[string]any{
		"multi_match": map[string]any{
			"query": f.value,
			"fields": []string{
				string(Title) + "^6", // 6 is 2 x the weight of short + long
				string(ShortDescription) + "^2",
				string(LongDescription),
			},
			"operator": "and",
		},
	}, nil
}

type Field string

func FieldFromString(s string) (Field, error) {
	_, ok := allFields[Field(s)]
	if !ok {
		return "", fmt.Errorf("field value %s is unsupported", s)
	}

	return Field(s), nil
}

// All the valid search fields for events
const (
	Filter               Field = "filter"
	GameID               Field = "gameId"
	Year                 Field = "year"
	Group                Field = "group"
	Title                Field = "title"
	ShortDescription     Field = "shortDescription"
	LongDescription      Field = "longDescription"
	EventType            Field = "eventType"
	GameSystem           Field = "gameSystem"
	RulesEdition         Field = "rulesEdition"
	MinPlayers           Field = "minPlayers"
	MaxPlayers           Field = "maxPlayers"
	AgeRequired          Field = "ageRequired"
	ExperienceRequired   Field = "experienceRequired"
	MaterialsProvided    Field = "materialsProvided"
	StartDateTime        Field = "startDateTime"
	Duration             Field = "duration"
	EndDateTime          Field = "endDateTime"
	GMNames              Field = "gmNames"
	Website              Field = "website"
	Email                Field = "email"
	Tournament           Field = "tournament"
	RoundNumber          Field = "roundNumber"
	TotalRounds          Field = "totalRounds"
	MinimumPlayTime      Field = "minimumPlayTime"
	AttendeeRegistration Field = "attendeeRegistration"
	Cost                 Field = "cost"
	Location             Field = "location"
	RoomName             Field = "roomName"
	TableNumber          Field = "tableNumber"
	SpecialCategory      Field = "specialCategory"
	TicketsAvailable     Field = "ticketsAvailable"
	LastModified         Field = "lastModified"
	AlsoRuns             Field = "alsoRuns"
	Prize                Field = "prize"
	RulesComplexity      Field = "rulesComplexity"
	OriginalOrder        Field = "originalOrder"
)

var (
	allFields = map[Field]any{
		Filter:               struct{}{},
		GameID:               struct{}{},
		Year:                 struct{}{},
		Group:                struct{}{},
		Title:                struct{}{},
		ShortDescription:     struct{}{},
		LongDescription:      struct{}{},
		EventType:            struct{}{},
		GameSystem:           struct{}{},
		RulesEdition:         struct{}{},
		MinPlayers:           struct{}{},
		MaxPlayers:           struct{}{},
		AgeRequired:          struct{}{},
		ExperienceRequired:   struct{}{},
		MaterialsProvided:    struct{}{},
		StartDateTime:        struct{}{},
		Duration:             struct{}{},
		EndDateTime:          struct{}{},
		GMNames:              struct{}{},
		Website:              struct{}{},
		Email:                struct{}{},
		Tournament:           struct{}{},
		RoundNumber:          struct{}{},
		TotalRounds:          struct{}{},
		MinimumPlayTime:      struct{}{},
		AttendeeRegistration: struct{}{},
		Cost:                 struct{}{},
		Location:             struct{}{},
		RoomName:             struct{}{},
		TableNumber:          struct{}{},
		SpecialCategory:      struct{}{},
		TicketsAvailable:     struct{}{},
		LastModified:         struct{}{},
		AlsoRuns:             struct{}{},
		Prize:                struct{}{},
		RulesComplexity:      struct{}{},
		OriginalOrder:        struct{}{},
	}
)

type OpType uint

const (
	And OpType = iota
	Or
)
