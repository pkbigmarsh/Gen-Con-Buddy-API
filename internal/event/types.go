package event

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	dateTimeFormat = "01/02/2006 03:04 PM"
	dateFormat     = "1/2/2006"
)

type Event struct {
	GameID               string       `json:"game_id"`
	Year                 int64        `json:"year"`
	Group                string       `json:"group"`
	Title                string       `json:"title"`
	ShortDescription     string       `json:"short_description"`
	LongDescription      string       `json:"long_description"`
	EventType            Type         `json:"event_type"`
	GameSystem           string       `json:"game_system"`
	RulesEdition         string       `json:"rules_edition"`
	MinPlayers           int64        `json:"min_players"`
	MaxPlayers           int64        `json:"max_players"`
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
	RoundNumber          int64        `json:"round_number"`
	TotalRounds          int64        `json:"total_rounds"`
	MinimumPlayTime      float64      `json:"minimum_play_time"`
	AttendeeRegistration Registration `json:"attendee_registration"`
	Cost                 float64      `json:"cost"`
	Location             string       `json:"location"`
	RoomName             string       `json:"room_name"`
	TableNumber          string       `json:"table_number"`
	SpecialCategory      Category     `json:"special_category"`
	TicketsAvailableTime int64        `json:"tickets_available"`
	LastModified         time.Time    `json:"last_modified"`
	AlsoRuns             time.Time    `json:"also_runs"`
	Prize                string       `json:"prize"`
	RulesComplexity      string       `json:"rules_complexity"`
	OriginalOrder        int64        `json:"original_order"`
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
		return fmt.Errorf("Unsupported field %s", field)
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
		return fmt.Errorf("Unsupported field %s", field)
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
		return fmt.Errorf("Unsupported field %s", field)
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
		return fmt.Errorf("Unsupported field %s", field)
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
		return fmt.Errorf("Unsupported field %s", field)
	}

	return nil
}
