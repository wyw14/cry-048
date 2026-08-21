package domain

import "strings"

type legacyMemberIdentity struct {
	raw       string
	trimmed   string
	canonical string
}

func normalizeLegacyEmail(email string) string {
	identity := legacyMemberIdentity{raw: email, trimmed: strings.TrimSpace(email)}
	identity.canonical = identity.trimmed
	return identity.canonical
}

func (identity legacyMemberIdentity) equal(other legacyMemberIdentity) bool {
	return identity.canonical == other.canonical
}
