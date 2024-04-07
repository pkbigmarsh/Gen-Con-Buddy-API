package event

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gencon_buddy_api/gcbapi"
)

const (
	dateTimeFormat = "01/02/2006 03:04 PM"
	dateFormat     = "1/2/2006"
)

type Event struct {
	GameID               string       `json:"gameId"`
	Year                 int64        `json:"year"`
	Group                string       `json:"group"`
	Title                string       `json:"title"`
	ShortDescription     string       `json:"shortDescription"`
	LongDescription      string       `json:"longDescription"`
	EventType            Type         `json:"eventType"`
	GameSystem           string       `json:"gameSystem"`
	RulesEdition         string       `json:"rulesEdition"`
	MinPlayers           int64        `json:"minPlayers"`
	MaxPlayers           int64        `json:"maxPlayers"`
	AgeRequired          AgeGroup     `json:"ageRequired"`
	ExperienceRequired   EXP          `json:"experienceRequired"`
	MaterialsProvided    string       `json:"materialsProvided"`
	StartDateTime        time.Time    `json:"startDateTime"`
	Duration             float64      `json:"duration"`
	EndDateTime          time.Time    `json:"endDateTime"`
	GMNames              string       `json:"gmNames"`
	Website              string       `json:"website"`
	Email                string       `json:"email"`
	Tournament           string       `json:"tournament"`
	RoundNumber          int64        `json:"roundNumber"`
	TotalRounds          int64        `json:"totalRounds"`
	MinimumPlayTime      float64      `json:"minimumPlayTime"`
	AttendeeRegistration Registration `json:"attendeeRegistration"`
	Cost                 float64      `json:"cost"`
	Location             string       `json:"location"`
	RoomName             string       `json:"roomName"`
	TableNumber          string       `json:"tableNumber"`
	SpecialCategory      Category     `json:"specialCategory"`
	TicketsAvailableTime int64        `json:"ticketsAvailable"`
	LastModified         time.Time    `json:"lastModified"`
	AlsoRuns             time.Time    `json:"alsoRuns"`
	Prize                string       `json:"prize"`
	RulesComplexity      string       `json:"rulesComplexity"`
	OriginalOrder        int64        `json:"originalOrder"`
}

func (e *Event) SetFieldFromString(field, value string) error {
	if value == "" {
		return nil
	}

	switch field {
	case "game_id":
		return e.setStringField(field, value)
	case "year":
		return e.setIntFieldFromString(field, value)
	case "group":
		return e.setStringField(field, value)
	case "title":
		return e.setStringField(field, value)
	case "short_description":
		return e.setStringField(field, value)
	case "long_description":
		return e.setStringField(field, value)
	case "event_type":
		if err := ValidateType(value); err != nil {
			return err
		}

		e.EventType = Type(value)
	case "game_system":
		return e.setStringField(field, value)
	case "rules_edition":
		return e.setStringField(field, value)
	case "min_players":
		return e.setIntFieldFromString(field, value)
	case "max_players":
		return e.setIntFieldFromString(field, value)
	case "age_required":
		if err := ValidateAgeGroup(value); err != nil {
			return err
		}

		e.AgeRequired = AgeGroup(value)
	case "experience_required":
		if err := ValidateEXP(value); err != nil {
			return err
		}

		e.ExperienceRequired = EXP(value)
	case "materials_provided":
		return e.setStringField(field, value)
	case "start_date_time":
		return e.setTimeFieldFromString(field, value)
	case "duration":
		return e.setFloatFieldFromString(field, value)
	case "end_date_time":
		return e.setTimeFieldFromString(field, value)
	case "gm_names":
		return e.setStringField(field, value)
	case "website":
		return e.setStringField(field, value)
	case "email":
		return e.setStringField(field, value)
	case "tournament":
		return e.setStringField(field, value)
	case "round_number":
		return e.setIntFieldFromString(field, value)
	case "total_rounds":
		return e.setIntFieldFromString(field, value)
	case "minimum_play_time":
		return e.setFloatFieldFromString(field, value)
	case "attendee_registration":
		if err := ValidateRegistration(value); err != nil {
			return err
		}

		e.AttendeeRegistration = Registration(value)
	case "cost":
		return e.setFloatFieldFromString(field, value)
	case "location":
		return e.setStringField(field, value)
	case "room_name":
		return e.setStringField(field, value)
	case "table_number":
		return e.setStringField(field, value)
	case "special_category":
		if err := ValidateCategory(value); err != nil {
			return err
		}

		e.SpecialCategory = Category(value)
	case "tickets_available":
		return e.setIntFieldFromString(field, value)
	case "last_modified":
		return e.setTimeFieldFromString(field, value)
	case "also_runs":
		return e.setTimeFieldFromString(field, value)
	case "prize":
		return e.setStringField(field, value)
	case "rules_complexity":
		return e.setStringField(field, value)
	case "original_order":
		return e.setIntFieldFromString(field, value)
	default:
		return fmt.Errorf("unsupported field %s", field)
	}

	return nil
}

func (e *Event) setStringField(field, value string) error {
	switch field {
	case "game_id":
		e.GameID = value
	case "group":
		e.Group = value
	case "title":
		e.Title = value
	case "short_description":
		e.ShortDescription = value
	case "long_description":
		e.LongDescription = value
	case "game_system":
		e.GameSystem = value
	case "rules_edition":
		e.RulesEdition = value
	case "materials_provided":
		e.MaterialsProvided = value
	case "gm_names":
		e.GMNames = value
	case "website":
		e.Website = value
	case "email":
		e.Email = value
	case "tournament":
		e.Tournament = value
	case "location":
		e.Location = value
	case "room_name":
		e.RoomName = value
	case "table_number":
		e.TableNumber = value
	case "prize":
		e.Prize = value
	case "rules_complexity":
		e.RulesComplexity = value
	default:
		return fmt.Errorf("unsupported field %s", field)
	}

	return nil
}

func (e *Event) setIntFieldFromString(field, value string) error {
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	switch field {
	case "year":
		e.Year = intValue
	case "min_players":
		e.MinPlayers = intValue
	case "max_players":
		e.MaxPlayers = intValue
	case "round_number":
		e.RoundNumber = intValue
	case "total_rounds":
		e.TotalRounds = intValue
	case "tickets_available":
		e.TicketsAvailableTime = intValue
	case "original_order":
		e.OriginalOrder = intValue
	default:
		return fmt.Errorf("unsupported field %s", field)
	}

	return nil
}

func (e *Event) setTimeFieldFromString(field, value string) error {
	var (
		timeValue time.Time
		err       error
	)

	if strings.HasSuffix(field, "date_time") {
		timeValue, err = time.Parse(dateTimeFormat, value)
	} else {
		timeValue, err = time.Parse(dateFormat, value)
	}

	if err != nil {
		return err
	}

	indy, err := time.LoadLocation("America/Indianapolis")
	if err != nil {
		return err
	}

	timeValue = timeValue.In(indy)

	switch field {
	case "start_date_time":
		e.StartDateTime = timeValue
	case "end_date_time":
		e.EndDateTime = timeValue
	case "last_modified":
		e.LastModified = timeValue
	case "also_runs":
		e.AlsoRuns = timeValue
	default:
		return fmt.Errorf("unsupported field %s", field)
	}

	return nil
}

func (e *Event) setFloatFieldFromString(field, value string) error {
	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	switch field {
	case "duration":
		e.Duration = floatVal
	case "minimum_play_time":
		e.MinimumPlayTime = floatVal
	case "cost":
		e.Cost = floatVal
	default:
		return fmt.Errorf("unsupported field %s", field)
	}

	return nil
}

// Externalize converts the internal Event shape into an api [gcbapi.Event]
func (e *Event) Externalize() gcbapi.Event {
	return gcbapi.Event{
		ID:   e.GameID,
		Type: "event",
		Attributes: gcbapi.EventAttributes{
			GameID:               e.GameID,
			Year:                 e.Year,
			Group:                e.Group,
			Title:                e.Title,
			ShortDescription:     e.ShortDescription,
			LongDescription:      e.LongDescription,
			EventType:            string(e.EventType),
			GameSystem:           e.GameSystem,
			RulesEdition:         e.RulesEdition,
			MinPlayers:           e.MinPlayers,
			MaxPlayers:           e.MaxPlayers,
			AgeRequired:          string(e.AgeRequired),
			ExperienceRequired:   string(e.ExperienceRequired),
			MaterialsProvided:    e.MaterialsProvided,
			StartDateTime:        e.StartDateTime,
			Duration:             e.Duration,
			EndDateTime:          e.EndDateTime,
			GMNames:              e.GMNames,
			Website:              e.Website,
			Email:                e.Email,
			Tournament:           e.Tournament,
			RoundNumber:          e.RoundNumber,
			TotalRounds:          e.TotalRounds,
			MinimumPlayTime:      e.MinimumPlayTime,
			AttendeeRegistration: string(e.AttendeeRegistration),
			Cost:                 e.Cost,
			Location:             e.Location,
			RoomName:             e.RoomName,
			TableNumber:          e.TableNumber,
			SpecialCategory:      string(e.SpecialCategory),
			TicketsAvailableTime: e.TicketsAvailableTime,
			LastModified:         e.LastModified,
			AlsoRuns:             e.AlsoRuns,
			Prize:                e.Prize,
			RulesComplexity:      e.RulesComplexity,
			OriginalOrder:        e.OriginalOrder,
		},
	}
}

func FromExternal(e gcbapi.Event) (*Event, error) {
	evt := &Event{
		GameID:               e.Attributes.GameID,
		Year:                 e.Attributes.Year,
		Group:                e.Attributes.Group,
		Title:                e.Attributes.Title,
		ShortDescription:     e.Attributes.ShortDescription,
		LongDescription:      e.Attributes.LongDescription,
		GameSystem:           e.Attributes.GameSystem,
		RulesEdition:         e.Attributes.RulesEdition,
		MinPlayers:           e.Attributes.MinPlayers,
		MaxPlayers:           e.Attributes.MaxPlayers,
		MaterialsProvided:    e.Attributes.MaterialsProvided,
		StartDateTime:        e.Attributes.StartDateTime,
		Duration:             e.Attributes.Duration,
		EndDateTime:          e.Attributes.EndDateTime,
		GMNames:              e.Attributes.GMNames,
		Website:              e.Attributes.Website,
		Email:                e.Attributes.Email,
		Tournament:           e.Attributes.Tournament,
		RoundNumber:          e.Attributes.RoundNumber,
		TotalRounds:          e.Attributes.TotalRounds,
		MinimumPlayTime:      e.Attributes.MinimumPlayTime,
		Cost:                 e.Attributes.Cost,
		Location:             e.Attributes.Location,
		RoomName:             e.Attributes.RoomName,
		TableNumber:          e.Attributes.TableNumber,
		TicketsAvailableTime: e.Attributes.TicketsAvailableTime,
		LastModified:         e.Attributes.LastModified,
		AlsoRuns:             e.Attributes.AlsoRuns,
		Prize:                e.Attributes.Prize,
		RulesComplexity:      e.Attributes.RulesComplexity,
		OriginalOrder:        e.Attributes.OriginalOrder,
	}

	if err := ValidateType(e.Attributes.EventType); err != nil {
		return nil, fmt.Errorf("invalided event type for event %s: %w", e.ID, err)
	}

	evt.EventType = Type(e.Attributes.EventType)

	if err := ValidateAgeGroup(e.Attributes.AgeRequired); err != nil {
		return nil, fmt.Errorf("invalided event age required for event %s: %w", e.ID, err)
	}

	evt.AgeRequired = AgeGroup(e.Attributes.AgeRequired)

	if err := ValidateEXP(e.Attributes.ExperienceRequired); err != nil {
		return nil, fmt.Errorf("invalided event experience required for event %s: %w", e.ID, err)
	}

	evt.ExperienceRequired = EXP(e.Attributes.ExperienceRequired)

	if err := ValidateRegistration(e.Attributes.AttendeeRegistration); err != nil {
		return nil, fmt.Errorf("invalided event registration for event %s: %w", e.ID, err)
	}

	evt.AttendeeRegistration = Registration(e.Attributes.AttendeeRegistration)

	if err := ValidateCategory(e.Attributes.EventType); err != nil {
		return nil, fmt.Errorf("invalided event type for event %s: %w", e.ID, err)
	}

	evt.SpecialCategory = Category(e.Attributes.EventType)

	return evt, nil
}
