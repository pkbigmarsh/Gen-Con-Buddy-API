package event

import (
	"fmt"
)

// AgeGroup is an enum for the limit set of Age Groups allowed for gencon events
type AgeGroup string

// All possible values for the enum AgeGroup
const (
	Kids     AgeGroup = "kids only (12 and under)"
	Everyone AgeGroup = "Everyone (6+)"
	Teen     AgeGroup = "Teen (13+)"
	Mature   AgeGroup = "Mature (18+)"
	Drinking AgeGroup = "21+"
)

// ValidateAgeGroup validates the incoming string against the defined enum list
func ValidateAgeGroup(v string) error {
	switch AgeGroup(v) {
	case Kids:
		return nil
	case Everyone:
		return nil
	case Teen:
		return nil
	case Mature:
		return nil
	case Drinking:
		return nil
	default:
		return fmt.Errorf("invalid value for Age Group: %s", v)
	}
}

// AgeGroupFromSearchTerm converts the searchable string value of the AgeGroup into the enum constant
func AgeGroupFromSearchTerm(s string) AgeGroup {
	switch s {
	case "kids":
		return Kids
	case "teen":
		return Teen
	case "mature":
		return Mature
	case "drinking":
		return Drinking
	default:
		return Everyone
	}
}

type Type string

const (
	SEM Type = "SEM - Seminar"
	ZED Type = "ZED - Isle of Misfit Events"
	ENT Type = "ENT - Entertainment Events"
	RPG Type = "RPG - Role Playing Game"
	BGM Type = "BGM - Board Game"
	CGM Type = "CGM - Non-Collectible / Tradable Card Game"
	WKS Type = "WKS - Workshop"
	MHE Type = "MHE - Miniature Hobby Events"
	LRP Type = "LRP - LARP"
	TRD Type = "TRD - Trade Day Event"
	HMN Type = "HMN - Historical Miniatures"
	NMN Type = "NMN - Non-Historical Miniatures"
	TCG Type = "TCG - Tradable Card Game"
	FLM Type = "FLM - Film Fest"
	KID Type = "KID - Kids Activities"
	ANI Type = "ANI - Anime Activities"
	TDA Type = "TDA - True Dungeon Adventures!"
	SPA Type = "SPA - Supplemental Activities"
	EGM Type = "EGM - Electronic Games"
)

// ValidateType validates the incoming string against the defined enum list
func ValidateType(v string) error {
	switch Type(v) {
	case SEM, ZED, ENT, RPG, BGM, CGM, WKS, MHE, LRP,
		TRD, HMN, NMN, TCG, FLM, KID, ANI, TDA, SPA, EGM:
		return nil
	default:
		return fmt.Errorf("invalid value for Type: %s", v)
	}
}

func EventTypeFromSearchTerm(s string) Type {
	switch s {
	case "SEM":
		return SEM
	case "ZED":
		return ZED
	case "ENT":
		return ENT
	case "RPG":
		return RPG
	case "BGM":
		return BGM
	case "CGM":
		return CGM
	case "WKS":
		return WKS
	case "MHE":
		return MHE
	case "LRP":
		return LRP
	case "TRD":
		return TRD
	case "HMN":
		return HMN
	case "NMN":
		return NMN
	case "TCG":
		return TCG
	case "FLM":
		return FLM
	case "KID":
		return KID
	case "ANI":
		return ANI
	case "TDA":
		return TDA
	case "SPA":
		return SPA
	case "EGM":
		return EGM
	default:
		return Type("invalid")
	}
}

// EXP is the experience enum
type EXP string

const (
	None   EXP = "None (You've never played before - rules will be taught)"
	Some   EXP = "Some (You've played it a bit and understand the basics)"
	Expert EXP = "Expert (You play it regularly and know all the rules)"
)

// ValidateEXP validates the incoming string against the defined enum list
func ValidateEXP(v string) error {
	switch EXP(v) {
	case None, Some, Expert:
		return nil
	default:
		return fmt.Errorf("invalid value for EXP: %s", v)
	}
}

func EXPFromSearchTerm(s string) EXP {
	switch s {
	case "none":
		return None
	case "some":
		return Some
	case "expert":
		return Expert
	default:
		return EXP("invalid")
	}
}

type Registration string

const (
	Open     Registration = "Yes, they can register for this round without having played in any other events"
	Free     Registration = "No, this event does not require tickets!"
	VIG      Registration = "VIG-only!"
	Invite   Registration = "No, this event is invite-only."
	Generic  Registration = "No, this is a generic ticket-only event!"
	TradeDay Registration = "Trade Day only!"
)

// ValidateRegistration validates the incoming string against the defined enum list
func ValidateRegistration(v string) error {
	switch Registration(v) {
	case Open, Free, VIG, Invite, Generic, TradeDay:
		return nil
	default:
		return fmt.Errorf("invalid value for Registration: %s", v)
	}
}

func RegistrationFromSearchTerm(s string) Registration {
	switch s {
	case "open":
		return Open
	case "free":
		return Free
	case "vig":
		return VIG
	case "invite":
		return Invite
	case "generic":
		return Generic
	case "tradeDay":
		return TradeDay
	default:
		return Registration("invalid")
	}
}

type Category string

const (
	No       Category = "none"
	Official Category = "Gen Con presents"
	Premier  Category = "Premier Event"
)

// ValidateCategory validates the incoming string against the defined enum list
func ValidateCategory(v string) error {
	switch Category(v) {
	case No, Official, Premier:
		return nil
	default:
		return fmt.Errorf("invalid value for Category: %s", v)
	}
}

func CategoryFromSearchTerm(s string) Category {
	switch s {
	case "none":
		return No
	case "official":
		return Official
	case "premier":
		return Premier
	default:
		return Category("invalid")
	}
}
