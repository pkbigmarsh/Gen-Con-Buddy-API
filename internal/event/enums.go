package event

// AgeGroup is an enum for the limit set of Age Groups allowed for gencon events
type AgeGroup string

// All possible values for the enum AgeGroup
const (
	Kids     AgeGroup = "Kids only (12 and under)"
	Everyone AgeGroup = "Everyone (6+)"
	Teen     AgeGroup = "Teen (13+)"
	Mature   AgeGroup = "Mature (18+)"
	Drinking AgeGroup = "21+"
)

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
)

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
	Open   Registration = "Yes, they can register for this round without having played in any other events"
	Free   Registration = "No, this event does not require tickets!"
	VIG    Registration = "VIG-only!"
	Invite Registration = "No, this event is invite-only."
)

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
	default:
		return Registration("invalid")
	}
}

type Category string

const (
	No       Category = "None"
	Official Category = "Gen Con presents"
	Premier  Category = "Premier Event"
)

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
