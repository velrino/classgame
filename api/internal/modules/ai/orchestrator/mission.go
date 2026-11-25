package orchestrator

import (
	"context"
	"log/slog"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/ai/agents/mission"
)

func (service *Service) BuildMission(
	ctx context.Context,
	input ai.BuildMissionInput,
) (ai.MissionBlueprint, error) {
	normalizedInput := normalizeInput(input)
	reportMissionProgress(normalizedInput.Progress, ai.MissionProgress{
		Stage:   "validating",
		Message: "Checking the mission request.",
		Percent: 5,
	})
	if blocked, reason := missionSafetyBlockReason(normalizedInput, ai.EducationGuardrail{Allowed: true}); blocked {
		reportMissionProgress(normalizedInput.Progress, ai.MissionProgress{
			Stage:   "guardrail",
			Message: "The request needs safer classroom material.",
			Percent: 100,
		})
		return buildGuardrailFallbackBlueprint(normalizedInput, reason), nil
	}

	if service.genAI == nil || !service.genAI.IsEnabled() {
		reportMissionProgress(normalizedInput.Progress, ai.MissionProgress{
			Stage:   "fallback",
			Message: "GenAI is not configured; using local fallback.",
			Percent: 100,
		})
		return buildFallbackBlueprint(normalizedInput, "genai is not configured"), nil
	}

	missionInput, summaryUsage := service.prepareMissionInput(ctx, normalizedInput)
	reportMissionProgress(normalizedInput.Progress, ai.MissionProgress{
		Stage:   "generating_mission",
		Message: "Building the playable mission preview.",
		Percent: 72,
	})

	blueprint, result, err := mission.Run(ctx, service.genAI, missionInput)
	if err != nil {
		slog.Warn("class mission genai fallback", "error", err)
		reportMissionProgress(normalizedInput.Progress, ai.MissionProgress{
			Stage:   "fallback",
			Message: "The model timed out; using local fallback.",
			Percent: 100,
		})
		return buildFallbackBlueprint(normalizedInput, err.Error()), nil
	}

	reportMissionProgress(normalizedInput.Progress, ai.MissionProgress{
		Stage:   "normalizing",
		Message: "Checking the mission preview.",
		Percent: 90,
	})
	blueprint = normalizeBlueprint(missionInput, blueprint)
	fallbackReason := blueprint.FallbackReason
	blueprint.GeneratedByAI = true
	blueprint.GeneratedByMock = false
	blueprint.Provider = result.Provider
	blueprint.Model = result.Model
	blueprint.ModelVersion = result.ModelVersion
	blueprint.FallbackReason = fallbackReason
	blueprint.Usage = addGenAIUsage(summaryUsage, ai.GenAIUsage{
		PromptTokenCount:     result.PromptTokenCount,
		CandidatesTokenCount: result.CandidatesTokenCount,
		ThoughtsTokenCount:   result.ThoughtsTokenCount,
		TotalTokenCount:      result.TotalTokenCount,
	})
	reportMissionProgress(normalizedInput.Progress, ai.MissionProgress{
		Stage:   "done",
		Message: "Mission preview ready.",
		Percent: 100,
	})

	return blueprint, nil
}

func reportMissionProgress(reporter ai.MissionProgressReporter, progress ai.MissionProgress) {
	if reporter == nil {
		return
	}
	if progress.Percent < 0 {
		progress.Percent = 0
	}
	if progress.Percent > 100 {
		progress.Percent = 100
	}
	reporter(progress)
}
