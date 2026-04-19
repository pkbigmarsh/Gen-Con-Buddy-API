package event

import (
	"fmt"
	"strings"

	"github.com/gencon_buddy_api/internal/search"
)

// SearchRequest contains all the needed information for searching
// for events
type SearchRequest struct {
	Terms     []search.Term
	Page      int
	Limit     int
	SortField Field
	SortDir   string
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
	// keywords that are specific enum types
	case AgeRequired:
		return search.NewKeywordSingle(f, string(AgeGroupFromSearchTerm(value)))
	case EventType:
		return search.NewKeywordSingle(f, string(EventTypeFromSearchTerm(value)))
	case ExperienceRequired:
		return search.NewKeywordSingle(f, string(EXPFromSearchTerm(value)))
	case AttendeeRegistration:
		return search.NewKeywordSingle(f, string(RegistrationFromSearchTerm(value)))
	case SpecialCategory:
		return search.NewKeywordSingle(f, string(CategoryFromSearchTerm(value)))
	// Keywords that have no special consideration
	case GameID, MaterialsRequired:
		return search.NewKeyword(f, value)
	// integer
	case Year, MinPlayers, MaxPlayers, RoundNumber,
		TotalRounds, TicketsAvailable, TotalTickets:
		return search.NewNumber(f, value)
	// Generic full text search fields
	case Group, Title, ShortDescription, LongDescription, GameSystem,
		RulesEdition, MaterialsProvided, MaterialsRequiredDetails, GMNames, Tournament,
		Location, RoomName, TableNumber, Prize, RulesComplexity:
		return search.NewText(f, value)
	// Full text search fields that have a subfield as well
	case Email, Website:
		text, err := search.NewText(f, value)
		if err != nil {
			return nil, err
		}

		// search on subfield stop, which breaks tokens on all special characters
		stopText, err := search.NewText(fmt.Sprintf("%s.stop", f), value)
		if err != nil {
			return nil, err
		}

		return search.NewBool().Should(text, stopText), err
	// double
	case Duration, MinimumPlayTime, Cost:
		return search.NewNumber(f, value)
	// Date searches
	case StartDateTime, EndDateTime, LastModified, AlsoRuns:
		return search.NewDate(f, value)
	// Special filter
	case Filter:
		return FilterTerm{value: value}, nil
	default:
		return nil, fmt.Errorf("Field %s is not supported as a search field", f)
	}
}

// ParseSort parses a "{field}.{asc|desc}" sort string.
// Returns the validated Field, direction, and any parse/validation error.
// The virtual "filter" field is not sortable.
func ParseSort(s string) (Field, string, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("sort must be formatted as {field}.{asc|desc}, got %q", s)
	}

	fieldStr, dir := parts[0], parts[1]

	if dir != "asc" && dir != "desc" {
		return "", "", fmt.Errorf("sort direction must be asc or desc, got %q", dir)
	}

	field, err := FieldFromString(fieldStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid sort field: %w", err)
	}

	if field == Filter {
		return "", "", fmt.Errorf("filter is a virtual field and cannot be used for sorting")
	}

	return field, dir, nil
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
	Filter                   Field = "filter"
	GameID                   Field = "gameId"
	Year                     Field = "year"
	Group                    Field = "group"
	Title                    Field = "title"
	ShortDescription         Field = "shortDescription"
	LongDescription          Field = "longDescription"
	EventType                Field = "eventType"
	GameSystem               Field = "gameSystem"
	RulesEdition             Field = "rulesEdition"
	MinPlayers               Field = "minPlayers"
	MaxPlayers               Field = "maxPlayers"
	AgeRequired              Field = "ageRequired"
	ExperienceRequired       Field = "experienceRequired"
	MaterialsProvided        Field = "materialsProvided"
	MaterialsRequired        Field = "materialsRequired"
	MaterialsRequiredDetails Field = "materialsRequiredDetails"
	StartDateTime            Field = "startDateTime"
	Duration                 Field = "duration"
	EndDateTime              Field = "endDateTime"
	GMNames                  Field = "gmNames"
	Website                  Field = "website"
	Email                    Field = "email"
	Tournament               Field = "tournament"
	RoundNumber              Field = "roundNumber"
	TotalRounds              Field = "totalRounds"
	MinimumPlayTime          Field = "minimumPlayTime"
	AttendeeRegistration     Field = "attendeeRegistration"
	Cost                     Field = "cost"
	Location                 Field = "location"
	RoomName                 Field = "roomName"
	TableNumber              Field = "tableNumber"
	SpecialCategory          Field = "specialCategory"
	TicketsAvailable         Field = "ticketsAvailable"
	TotalTickets             Field = "totalTickets"
	LastModified             Field = "lastModified"
	AlsoRuns                 Field = "alsoRuns"
	Prize                    Field = "prize"
	RulesComplexity          Field = "rulesComplexity"
	OriginalOrder            Field = "originalOrder"
)

var (
	allFields = map[Field]any{
		Filter:                   struct{}{},
		GameID:                   struct{}{},
		Year:                     struct{}{},
		Group:                    struct{}{},
		Title:                    struct{}{},
		ShortDescription:         struct{}{},
		LongDescription:          struct{}{},
		EventType:                struct{}{},
		GameSystem:               struct{}{},
		RulesEdition:             struct{}{},
		MinPlayers:               struct{}{},
		MaxPlayers:               struct{}{},
		AgeRequired:              struct{}{},
		ExperienceRequired:       struct{}{},
		MaterialsProvided:        struct{}{},
		MaterialsRequired:        struct{}{},
		MaterialsRequiredDetails: struct{}{},
		StartDateTime:            struct{}{},
		Duration:                 struct{}{},
		EndDateTime:              struct{}{},
		GMNames:                  struct{}{},
		Website:                  struct{}{},
		Email:                    struct{}{},
		Tournament:               struct{}{},
		RoundNumber:              struct{}{},
		TotalRounds:              struct{}{},
		MinimumPlayTime:          struct{}{},
		AttendeeRegistration:     struct{}{},
		Cost:                     struct{}{},
		Location:                 struct{}{},
		RoomName:                 struct{}{},
		TableNumber:              struct{}{},
		SpecialCategory:          struct{}{},
		TicketsAvailable:         struct{}{},
		TotalTickets:             struct{}{},
		LastModified:             struct{}{},
		AlsoRuns:                 struct{}{},
		Prize:                    struct{}{},
		RulesComplexity:          struct{}{},
		OriginalOrder:            struct{}{},
	}
)

type OpType uint

const (
	And OpType = iota
	Or
)
