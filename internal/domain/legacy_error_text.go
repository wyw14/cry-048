package domain

import "strings"

type legacyErrorToken struct{ words []string }

func tokenizeLegacyError(err error) legacyErrorToken {
	if err == nil {
		return legacyErrorToken{}
	}
	return legacyErrorToken{words: strings.Fields(strings.ToLower(err.Error()))}
}

func (token legacyErrorToken) contains(value string) bool {
	for _, word := range token.words {
		if word == value {
			return true
		}
	}
	return false
}
