package students

import (
	"strings"
)

func truncateStudentText(value string, limit int) string {
	value = normalizeText(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}

	return strings.TrimSpace(string(runes[:limit]))
}

func normalizeMissionCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeClassCode(value string) string {
	return strings.TrimSpace(value)
}

func normalizeStudentName(value string) string {
	return truncateStudentText(value, studentNameMaxLength)
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
