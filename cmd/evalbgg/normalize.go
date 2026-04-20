package main

import (
	"strings"
	"unicode"
)

var titleStopwords = map[string]bool{
	"tournament": true, "finals": true, "final": true, "qualifier": true,
	"round": true, "semi": true, "beginner": true, "beginners": true,
	"experienced": true, "advanced": true, "mini": true, "open": true,
	"championship": true, "preliminary": true,
	"event": true, "demo": true, "intro": true, "introduction": true,
	"teach": true, "teaching": true, "with": true, "for": true,
	"to": true, "the": true, "a": true, "an": true, "of": true,
	"in": true, "and": true, "by": true, "at": true, "upgraded": true,
	"components": true, "expansion": true,
}

var genericEditionTerms = map[string]bool{
	"1st": true, "2nd": true, "3rd": true, "4th": true, "5th": true,
	"first": true, "second": true, "third": true, "revised": true,
	"standard": true, "deluxe": true, "basic": true, "classic": true,
}

// normalize lowercases s, strips punctuation (keeping & and alphanumerics),
// and collapses whitespace.
func normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if r == '&' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// isInformativeEdition returns true if edition contains tokens beyond bare ordinals.
func isInformativeEdition(edition string) bool {
	for _, w := range strings.Fields(normalize(edition)) {
		if !genericEditionTerms[w] {
			return true
		}
	}
	return false
}

// extractTitleDerived strips game system tokens and title stopwords from title,
// returning the remaining edition-like tokens joined by spaces.
func extractTitleDerived(gameSystem, title string) string {
	sysTokens := make(map[string]bool)
	for _, w := range strings.Fields(normalize(gameSystem)) {
		sysTokens[w] = true
	}

	var result []string
	for _, w := range strings.Fields(normalize(title)) {
		if !sysTokens[w] && !titleStopwords[w] {
			result = append(result, w)
		}
	}
	return strings.Join(result, " ")
}
