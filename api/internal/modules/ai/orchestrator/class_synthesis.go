package orchestrator

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/agents/classsynthesis"
)

func (service *Service) BuildClassLearningSynthesis(
	ctx context.Context,
	input ai.BuildClassLearningSynthesisInput,
) (ai.ClassLearningSynthesis, error) {
	input = normalizeClassLearningSynthesisInput(input)
	fallback := buildFallbackClassLearningSynthesis(input, "")
	if len(input.Evidence) == 0 {
		fallback.FallbackReason = "No finalized student evidence is available."
		return fallback, nil
	}
	if service.genAI == nil || !service.genAI.IsEnabled() {
		fallback.FallbackReason = "genai is not configured"
		return fallback, nil
	}

	synthesis, result, err := classsynthesis.Run(ctx, service.genAI, input)
	if err != nil {
		slog.Warn("class learning synthesis genai fallback", "error", err)
		fallback.FallbackReason = truncateText(normalizeText(err.Error()), 240)
		return fallback, nil
	}

	synthesis = normalizeClassLearningSynthesis(input, synthesis)
	synthesis.GeneratedByAI = true
	synthesis.Provider = result.Provider
	synthesis.Model = result.Model
	synthesis.ModelVersion = result.ModelVersion
	synthesis.Usage = ai.GenAIUsage{
		PromptTokenCount:     result.PromptTokenCount,
		CandidatesTokenCount: result.CandidatesTokenCount,
		ThoughtsTokenCount:   result.ThoughtsTokenCount,
		TotalTokenCount:      result.TotalTokenCount,
	}

	return synthesis, nil
}

func normalizeClassLearningSynthesisInput(input ai.BuildClassLearningSynthesisInput) ai.BuildClassLearningSynthesisInput {
	input.ClassID = truncateText(normalizeText(input.ClassID), 80)
	if len(input.Evidence) > 100 {
		input.Evidence = input.Evidence[:100]
	}
	result := make([]ai.ClassLearningStudentEvidence, 0, len(input.Evidence))
	seen := make(map[string]struct{}, len(input.Evidence))
	for _, evidence := range input.Evidence {
		evidence.StudentID = truncateText(normalizeText(evidence.StudentID), 120)
		if evidence.StudentID == "" {
			continue
		}
		if _, exists := seen[evidence.StudentID]; exists {
			continue
		}
		seen[evidence.StudentID] = struct{}{}
		evidence.StudentName = truncateText(normalizeText(evidence.StudentName), 80)
		evidence.Strengths = normalizeLearningSignals(evidence.Strengths, 4)
		evidence.ImprovementAreas = normalizeLearningSignals(evidence.ImprovementAreas, 4)
		result = append(result, evidence)
	}
	input.Evidence = result

	return input
}

func buildFallbackClassLearningSynthesis(
	input ai.BuildClassLearningSynthesisInput,
	reason string,
) ai.ClassLearningSynthesis {
	strengths := clusterClassLabels(input.Evidence, true)
	needs := clusterClassLabels(input.Evidence, false)
	next := "Collect more finalized attempts before choosing the next whole-class adaptation."
	if len(needs) > 0 {
		next = "Begin the next mission with a short whole-class check on: " + needs[0].Label
	}

	return ai.ClassLearningSynthesis{
		CommonStrengths: strengths,
		CommonNeeds:     needs,
		NextAdaptation:  truncateText(next, 280),
		GeneratedByAI:   false,
		Provider:        "observed_class_evidence",
		FallbackReason:  truncateText(normalizeText(reason), 240),
	}
}

func normalizeClassLearningSynthesis(
	input ai.BuildClassLearningSynthesisInput,
	synthesis ai.ClassLearningSynthesis,
) ai.ClassLearningSynthesis {
	fallback := buildFallbackClassLearningSynthesis(input, "")
	allowedStudents := make(map[string]struct{}, len(input.Evidence))
	allowedLabels := make(map[string]struct{})
	for _, evidence := range input.Evidence {
		allowedStudents[evidence.StudentID] = struct{}{}
		for _, label := range append(append([]string{}, evidence.Strengths...), evidence.ImprovementAreas...) {
			allowedLabels[strings.ToLower(normalizeText(label))] = struct{}{}
		}
	}
	synthesis.CommonStrengths = normalizeClassClusters(synthesis.CommonStrengths, allowedStudents, allowedLabels)
	synthesis.CommonNeeds = normalizeClassClusters(synthesis.CommonNeeds, allowedStudents, allowedLabels)
	if len(synthesis.CommonStrengths) == 0 {
		synthesis.CommonStrengths = fallback.CommonStrengths
	}
	if len(synthesis.CommonNeeds) == 0 {
		synthesis.CommonNeeds = fallback.CommonNeeds
	}
	synthesis.NextAdaptation = truncateText(normalizeText(synthesis.NextAdaptation), 280)
	if synthesis.NextAdaptation == "" {
		synthesis.NextAdaptation = fallback.NextAdaptation
	}

	return synthesis
}

func normalizeClassClusters(
	clusters []ai.ClassLearningCluster,
	allowedStudents map[string]struct{},
	allowedLabels map[string]struct{},
) []ai.ClassLearningCluster {
	result := make([]ai.ClassLearningCluster, 0, min(len(clusters), 4))
	for _, cluster := range clusters {
		if len(result) >= 4 {
			break
		}
		label := truncateText(normalizeText(cluster.Label), 180)
		if label == "" {
			continue
		}
		studentIDs := filterAllowedIDs(cluster.StudentIDs, allowedStudents, 100)
		if len(studentIDs) == 0 {
			continue
		}
		evidenceLabels := make([]string, 0, len(cluster.EvidenceLabels))
		for _, evidenceLabel := range cluster.EvidenceLabels {
			evidenceLabel = truncateText(normalizeText(evidenceLabel), 180)
			if _, exists := allowedLabels[strings.ToLower(evidenceLabel)]; exists {
				evidenceLabels = append(evidenceLabels, evidenceLabel)
			}
		}
		result = append(result, ai.ClassLearningCluster{
			Label:          label,
			StudentIDs:     studentIDs,
			EvidenceLabels: normalizeLimitedStrings(evidenceLabels, 20, 180),
		})
	}

	return result
}

func clusterClassLabels(
	evidence []ai.ClassLearningStudentEvidence,
	strengths bool,
) []ai.ClassLearningCluster {
	clusters := make([]ai.ClassLearningCluster, 0)
	for _, student := range evidence {
		labels := student.ImprovementAreas
		if strengths {
			labels = student.Strengths
		}
		for _, label := range labels {
			label = truncateText(normalizeText(label), 180)
			if label == "" {
				continue
			}
			tokens := semanticTokens(label)
			matched := -1
			for index, cluster := range clusters {
				if semanticSimilarity(tokens, semanticTokens(cluster.Label)) >= 0.42 {
					matched = index
					break
				}
			}
			if matched < 0 {
				clusters = append(clusters, ai.ClassLearningCluster{
					Label:          label,
					StudentIDs:     []string{student.StudentID},
					EvidenceLabels: []string{label},
				})
				continue
			}
			clusters[matched].StudentIDs = appendUnique(clusters[matched].StudentIDs, student.StudentID)
			clusters[matched].EvidenceLabels = appendUnique(clusters[matched].EvidenceLabels, label)
		}
	}
	sort.SliceStable(clusters, func(left int, right int) bool {
		if len(clusters[left].StudentIDs) == len(clusters[right].StudentIDs) {
			return clusters[left].Label < clusters[right].Label
		}

		return len(clusters[left].StudentIDs) > len(clusters[right].StudentIDs)
	})
	if len(clusters) > 4 {
		clusters = clusters[:4]
	}

	return clusters
}
