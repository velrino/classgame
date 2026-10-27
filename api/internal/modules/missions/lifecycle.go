package missions

import (
	"time"

	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/tenancy"
)

const initialMissionRevision int64 = 1

func snapshotFromMission(mission MissionDocument, revision int64) MissionSnapshot {
	return MissionSnapshot{
		Revision:                 revision,
		OrganizationID:           mission.OrganizationID,
		DistrictID:               mission.DistrictID,
		SchoolID:                 mission.SchoolID,
		TeacherID:                mission.TeacherID,
		MissionName:              mission.MissionName,
		Title:                    mission.Title,
		Summary:                  mission.Summary,
		Subject:                  mission.Subject,
		GradeLevel:               mission.GradeLevel,
		LearningGoal:             mission.LearningGoal,
		TeacherPrompt:            mission.TeacherPrompt,
		AudienceType:             mission.AudienceType,
		TargetStudentIDs:         mission.TargetStudentIDs,
		Attachments:              mission.Attachments,
		PreviewQuestions:         mission.PreviewQuestions,
		KnowledgeChunks:          mission.KnowledgeChunks,
		SuggestedDurationMinutes: mission.SuggestedDurationMinutes,
		Stages:                   mission.Stages,
		Suggestions:              mission.Suggestions,
		PositiveSignals:          mission.PositiveSignals,
		ImprovementSignals:       mission.ImprovementSignals,
		EducationGuardrail:       mission.EducationGuardrail,
		MissionFlow:              mission.MissionFlow,
		Orchestration:            mission.Orchestration,
		UpdatedBy:                mission.UpdatedBy,
		UpdatedAt:                mission.UpdatedAt,
	}
}

func (mission *MissionDocument) applySnapshot(snapshot MissionSnapshot) {
	mission.MissionName = snapshot.MissionName
	mission.OrganizationID = snapshot.OrganizationID
	mission.DistrictID = snapshot.DistrictID
	mission.SchoolID = snapshot.SchoolID
	mission.TeacherID = snapshot.TeacherID
	mission.Title = snapshot.Title
	mission.Summary = snapshot.Summary
	mission.Subject = snapshot.Subject
	mission.GradeLevel = snapshot.GradeLevel
	mission.LearningGoal = snapshot.LearningGoal
	mission.TeacherPrompt = snapshot.TeacherPrompt
	mission.AudienceType = snapshot.AudienceType
	mission.TargetStudentIDs = snapshot.TargetStudentIDs
	mission.Attachments = snapshot.Attachments
	mission.PreviewQuestions = snapshot.PreviewQuestions
	mission.KnowledgeChunks = snapshot.KnowledgeChunks
	mission.SuggestedDurationMinutes = snapshot.SuggestedDurationMinutes
	mission.Stages = snapshot.Stages
	mission.Suggestions = snapshot.Suggestions
	mission.PositiveSignals = snapshot.PositiveSignals
	mission.ImprovementSignals = snapshot.ImprovementSignals
	mission.EducationGuardrail = snapshot.EducationGuardrail
	mission.MissionFlow = snapshot.MissionFlow
	mission.Orchestration = snapshot.Orchestration
	mission.UpdatedBy = snapshot.UpdatedBy
	mission.UpdatedAt = snapshot.UpdatedAt
	ensureMissionFlow(mission)
}

func hydrateMissionLifecycle(mission *MissionDocument) {
	hydrateMissionTenantScope(mission)

	if mission.Status == StatusPreview {
		mission.Status = StatusDraft
	}

	if mission.Draft == nil {
		revision := mission.DraftRevision
		if revision <= 0 {
			revision = initialMissionRevision
		}

		draft := snapshotFromMission(*mission, revision)
		mission.Draft = &draft
		mission.DraftRevision = revision
	}

	wasPublished := mission.Status == StatusPublished ||
		mission.StatusBeforeDelete == StatusPublished ||
		!mission.PublishedAt.IsZero()
	if mission.Published == nil && wasPublished {
		revision := mission.PublishedRevision
		if revision <= 0 {
			revision = mission.Draft.Revision
		}

		published := snapshotFromMission(*mission, revision)
		mission.Published = &published
		mission.PublishedRevision = revision
	}

	if mission.DraftRevision <= 0 {
		mission.DraftRevision = mission.Draft.Revision
	}
	if mission.Published != nil && mission.PublishedRevision <= 0 {
		mission.PublishedRevision = mission.Published.Revision
	}

	mission.HasUnpublishedChanges = mission.Published == nil ||
		mission.DraftRevision != mission.PublishedRevision
	ensureMissionFlow(mission)
}

func hydrateMissionTenantScope(mission *MissionDocument) {
	if mission == nil {
		return
	}

	scope := tenancy.NormalizeScope(tenancy.Scope{
		OrganizationID: mission.OrganizationID,
		DistrictID:     mission.DistrictID,
		SchoolID:       mission.SchoolID,
	})
	mission.OrganizationID = scope.OrganizationID
	mission.DistrictID = scope.DistrictID
	mission.SchoolID = scope.SchoolID
	if mission.TeacherID == "" {
		mission.TeacherID = mission.CreatedBy.ProviderUserID
	}
	hydrateSnapshotTenantScope(mission.Draft, mission)
	hydrateSnapshotTenantScope(mission.Published, mission)
}

func hydrateSnapshotTenantScope(
	snapshot *MissionSnapshot,
	mission *MissionDocument,
) {
	if snapshot == nil || mission == nil {
		return
	}
	if snapshot.OrganizationID == "" {
		snapshot.OrganizationID = mission.OrganizationID
	}
	if snapshot.DistrictID == "" {
		snapshot.DistrictID = mission.DistrictID
	}
	if snapshot.SchoolID == "" {
		snapshot.SchoolID = mission.SchoolID
	}
	if snapshot.TeacherID == "" {
		snapshot.TeacherID = mission.TeacherID
	}
}

func draftView(mission MissionDocument) MissionDocument {
	hydrateMissionLifecycle(&mission)
	if mission.Draft != nil {
		mission.applySnapshot(*mission.Draft)
	}
	if mission.Status != StatusDeleted {
		if mission.Published != nil && mission.DraftRevision == mission.PublishedRevision {
			mission.Status = StatusPublished
		} else {
			mission.Status = StatusDraft
		}
	}

	return mission
}

func publishedView(mission MissionDocument) (MissionDocument, bool) {
	hydrateMissionLifecycle(&mission)
	if mission.Status == StatusDeleted || mission.Published == nil {
		return MissionDocument{}, false
	}

	mission.applySnapshot(*mission.Published)
	mission.Status = StatusPublished
	mission.HasUnpublishedChanges = mission.DraftRevision != mission.PublishedRevision

	return mission, true
}

// PublishedView returns the student-safe current published snapshot. Draft
// edits remain private until the teacher publishes them.
func PublishedView(mission MissionDocument) (MissionDocument, bool) {
	return publishedView(mission)
}

// ViewAtPublishedRevision pins a student attempt to the immutable content that
// was live when it started, even if the teacher publishes a newer revision.
func ViewAtPublishedRevision(
	mission MissionDocument,
	published MissionPublishedRevisionDocument,
) (MissionDocument, bool) {
	if published.Revision <= 0 ||
		published.MissionID != "" && published.MissionID != mission.ID ||
		published.ClassID != "" && published.ClassID != mission.ClassID {
		return MissionDocument{}, false
	}

	mission.applySnapshot(published.Snapshot)
	mission.Published = &published.Snapshot
	mission.PublishedRevision = published.Revision
	mission.PublishedAt = published.PublishedAt
	mission.Status = StatusPublished
	mission.HasUnpublishedChanges = mission.DraftRevision != published.Revision

	return mission, true
}

func newSnapshotFromInput(input CreateMissionInput, revision int64) MissionSnapshot {
	mission := buildMissionDocumentFields(input)
	mission.UpdatedBy = input.Teacher
	mission.UpdatedAt = input.Timestamp

	return snapshotFromMission(mission, revision)
}

func nonZeroTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}

	return value
}

func ensureMissionFlow(mission *MissionDocument) {
	if mission == nil || len(mission.MissionFlow.Nodes) > 0 && len(mission.MissionFlow.Edges) > 0 {
		return
	}

	title := mission.Title
	if title == "" {
		title = mission.MissionName
	}
	if title == "" {
		title = "Class mission"
	}

	nodes := []generation.MissionFlowNode{
		{
			ID:          "start",
			Kind:        "start",
			Label:       "Teacher starts mission",
			Description: "The teacher provides the learning goal, prompt, and optional worksheet files.",
			Position:    generation.MissionFlowPosition{X: 0, Y: 0},
		},
		{
			ID:          "preview",
			Kind:        "setup",
			Label:       "Mia builds preview",
			Description: "Mia turns " + title + " into a short student-safe mission.",
			Position:    generation.MissionFlowPosition{X: 0, Y: 120},
		},
	}

	for index, stage := range mission.Stages {
		if index >= 3 {
			break
		}
		label := stage.Title
		if label == "" {
			label = "Mission stage"
		}
		description := stage.Goal
		if description == "" {
			description = "Students practice one part of the learning goal."
		}

		nodes = append(nodes, generation.MissionFlowNode{
			ID:          "stage-" + string(rune('1'+index)),
			Kind:        "activity",
			Label:       label,
			Description: description,
			Position:    generation.MissionFlowPosition{X: 0, Y: float64(240 + index*120)},
		})
	}

	decisionY := float64(240 + len(nodes)*90)
	nodes = append(nodes,
		generation.MissionFlowNode{
			ID:          "check-understanding",
			Kind:        "decision",
			Label:       "Understanding check",
			Description: "Mia checks whether the student can explain the answer and reasoning.",
			Position:    generation.MissionFlowPosition{X: 0, Y: decisionY},
		},
		generation.MissionFlowNode{
			ID:          "adaptive-feedback",
			Kind:        "feedback",
			Label:       "Adaptive feedback",
			Description: "Mia changes the explanation style when the student needs support.",
			Position:    generation.MissionFlowPosition{X: 260, Y: decisionY},
		},
		generation.MissionFlowNode{
			ID:          "complete",
			Kind:        "complete",
			Label:       "Mission complete",
			Description: "Progress and notebook signals are saved for the teacher.",
			Position:    generation.MissionFlowPosition{X: 0, Y: decisionY + 130},
		},
	)

	edges := make([]generation.MissionFlowEdge, 0, len(nodes))
	for index := 0; index < len(nodes)-4; index++ {
		edges = append(edges, generation.MissionFlowEdge{
			ID:     nodes[index].ID + "-to-" + nodes[index+1].ID,
			Source: nodes[index].ID,
			Target: nodes[index+1].ID,
			Label:  "Next",
		})
	}
	previousID := nodes[len(nodes)-4].ID
	edges = append(edges,
		generation.MissionFlowEdge{
			ID:     previousID + "-to-check-understanding",
			Source: previousID,
			Target: "check-understanding",
			Label:  "Check",
		},
		generation.MissionFlowEdge{
			ID:     "check-understanding-to-adaptive-feedback",
			Source: "check-understanding",
			Target: "adaptive-feedback",
			Label:  "Needs support",
		},
		generation.MissionFlowEdge{
			ID:     "adaptive-feedback-to-complete",
			Source: "adaptive-feedback",
			Target: "complete",
			Label:  "Ready",
		},
		generation.MissionFlowEdge{
			ID:     "check-understanding-to-complete",
			Source: "check-understanding",
			Target: "complete",
			Label:  "Understands",
		},
	)

	mission.MissionFlow = generation.MissionFlow{
		Nodes: nodes,
		Edges: edges,
	}
}
