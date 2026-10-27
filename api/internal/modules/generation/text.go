package generation

import (
	"strings"
)

func addGenAIUsage(left GenAIUsage, right GenAIUsage) GenAIUsage {
	return GenAIUsage{
		PromptTokenCount:     left.PromptTokenCount + right.PromptTokenCount,
		CandidatesTokenCount: left.CandidatesTokenCount + right.CandidatesTokenCount,
		ThoughtsTokenCount:   left.ThoughtsTokenCount + right.ThoughtsTokenCount,
		TotalTokenCount:      left.TotalTokenCount + right.TotalTokenCount,
	}
}

func truncateText(value string, maxLength int) string {
	if maxLength <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	if maxLength <= 1 {
		return string(runes[:maxLength])
	}

	return strings.TrimSpace(string(runes[:maxLength-1])) + "…"
}

func semanticTokens(value string) map[string]struct{} {
	stopwords := map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "the": {}, "to": {}, "of": {}, "in": {},
		"with": {}, "on": {}, "for": {}, "student": {}, "students": {}, "needs": {},
	}
	result := make(map[string]struct{})
	for _, token := range strings.Fields(strings.ToLower(value)) {
		token = strings.Trim(token, ".,:;!?()[]{}\"'")
		for _, suffix := range []string{"ing", "ed", "es", "s"} {
			if len(token) > len(suffix)+3 && strings.HasSuffix(token, suffix) {
				token = strings.TrimSuffix(token, suffix)
				break
			}
		}
		if token == "" {
			continue
		}
		if _, blocked := stopwords[token]; blocked {
			continue
		}
		result[token] = struct{}{}
	}

	return result
}

func semanticSimilarity(left map[string]struct{}, right map[string]struct{}) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	intersection := 0
	union := make(map[string]struct{}, len(left)+len(right))
	for token := range left {
		union[token] = struct{}{}
		if _, exists := right[token]; exists {
			intersection++
		}
	}
	for token := range right {
		union[token] = struct{}{}
	}

	return float64(intersection) / float64(len(union))
}

func normalizeLimitedStrings(values []string, limit int, maxLength int) []string {
	result := make([]string, 0, min(len(values), limit))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = truncateText(normalizeText(value), maxLength)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
		if len(result) >= limit {
			break
		}
	}

	return result
}

func filterAllowedIDs(values []string, allowed map[string]struct{}, limit int) []string {
	result := make([]string, 0, min(len(values), limit))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = normalizeFlowID(value)
		if value == "" {
			continue
		}
		if _, exists := allowed[value]; !exists {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
		if len(result) >= limit {
			break
		}
	}

	return result
}

func appendUnique(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}

	return append(values, value)
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
