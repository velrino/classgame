package generation

import (
	"fmt"
	"strings"
)

const maxMissionFlowNodes = 8

const maxMissionFlowEdges = 10

func normalizeMissionFlow(input BuildMissionInput, blueprint MissionBlueprint) MissionFlow {
	nodes := make([]MissionFlowNode, 0, len(blueprint.MissionFlow.Nodes))
	seenNodes := make(map[string]struct{}, len(blueprint.MissionFlow.Nodes))
	for _, node := range blueprint.MissionFlow.Nodes {
		if len(nodes) >= maxMissionFlowNodes {
			break
		}

		id := normalizeFlowID(node.ID)
		if id == "" {
			id = fmt.Sprintf("node-%d", len(nodes)+1)
		}
		if _, exists := seenNodes[id]; exists {
			continue
		}

		label := truncateText(normalizeText(node.Label), 64)
		if label == "" {
			continue
		}
		description := truncateText(normalizeText(node.Description), 180)
		if description == "" {
			description = "Mia guides the student through this mission step."
		}

		seenNodes[id] = struct{}{}
		nodes = append(nodes, MissionFlowNode{
			ID:          id,
			Kind:        normalizeFlowKind(node.Kind),
			Label:       label,
			Description: description,
			Position: MissionFlowPosition{
				X: clampFloat(node.Position.X, -800, 800),
				Y: clampFloat(node.Position.Y, -200, 1200),
			},
		})
	}
	if len(nodes) < 4 {
		return fallbackMissionFlow(input, blueprint)
	}

	edges := make([]MissionFlowEdge, 0, len(blueprint.MissionFlow.Edges))
	seenEdges := make(map[string]struct{}, len(blueprint.MissionFlow.Edges))
	for _, edge := range blueprint.MissionFlow.Edges {
		if len(edges) >= maxMissionFlowEdges {
			break
		}

		source := normalizeFlowID(edge.Source)
		target := normalizeFlowID(edge.Target)
		if source == "" || target == "" || source == target {
			continue
		}
		if _, exists := seenNodes[source]; !exists {
			continue
		}
		if _, exists := seenNodes[target]; !exists {
			continue
		}

		id := normalizeFlowID(edge.ID)
		if id == "" {
			id = source + "-to-" + target
		}
		if _, exists := seenEdges[id]; exists {
			continue
		}

		seenEdges[id] = struct{}{}
		edges = append(edges, MissionFlowEdge{
			ID:     id,
			Source: source,
			Target: target,
			Label:  truncateText(normalizeText(edge.Label), 48),
		})
	}
	if len(edges) < 3 {
		return fallbackMissionFlow(input, blueprint)
	}

	return MissionFlow{Nodes: nodes, Edges: edges}
}

func fallbackMissionFlow(input BuildMissionInput, blueprint MissionBlueprint) MissionFlow {
	title := normalizeText(blueprint.Title)
	if title == "" {
		title = input.MissionName
	}
	if title == "" {
		title = input.Subject + " mission"
	}

	nodes := []MissionFlowNode{
		{
			ID:          "start",
			Kind:        "start",
			Label:       "Teacher starts mission",
			Description: "The teacher provides the learning goal, prompt, and optional worksheet files.",
			Position:    MissionFlowPosition{X: 0, Y: 0},
		},
		{
			ID:          "generate-preview",
			Kind:        "setup",
			Label:       "Mia builds preview",
			Description: truncateText("Mia turns "+title+" into a short student-safe mission.", 180),
			Position:    MissionFlowPosition{X: 0, Y: 120},
		},
	}

	for index, stage := range blueprint.Stages {
		if index >= 3 {
			break
		}
		stageID := normalizeFlowID(stage.ID)
		if stageID == "" {
			stageID = fmt.Sprintf("stage-%d", index+1)
		}
		label := truncateText(normalizeText(stage.Title), 64)
		if label == "" {
			label = fmt.Sprintf("Mission stage %d", index+1)
		}
		description := truncateText(normalizeText(stage.Goal), 180)
		if description == "" {
			description = "Students practice one part of the learning goal."
		}

		nodes = append(nodes, MissionFlowNode{
			ID:          stageID,
			Kind:        "activity",
			Label:       label,
			Description: description,
			Position:    MissionFlowPosition{X: 0, Y: float64(240 + index*120)},
		})
	}

	decisionY := float64(240 + len(nodes)*90)
	nodes = append(nodes,
		MissionFlowNode{
			ID:          "check-understanding",
			Kind:        "decision",
			Label:       "Understanding check",
			Description: "Mia checks whether the student can explain the answer and the reasoning.",
			Position:    MissionFlowPosition{X: 0, Y: decisionY},
		},
		MissionFlowNode{
			ID:          "adaptive-feedback",
			Kind:        "feedback",
			Label:       "Adaptive feedback",
			Description: "If the student is stuck, Mia changes the explanation style and records the pattern.",
			Position:    MissionFlowPosition{X: 260, Y: decisionY},
		},
		MissionFlowNode{
			ID:          "complete",
			Kind:        "complete",
			Label:       "Mission complete",
			Description: "The student finishes with progress and notebook signals saved for the teacher.",
			Position:    MissionFlowPosition{X: 0, Y: decisionY + 130},
		},
	)

	edges := make([]MissionFlowEdge, 0, len(nodes)+1)
	for index := 0; index < len(nodes)-4; index++ {
		edges = append(edges, MissionFlowEdge{
			ID:     nodes[index].ID + "-to-" + nodes[index+1].ID,
			Source: nodes[index].ID,
			Target: nodes[index+1].ID,
			Label:  "Next",
		})
	}
	previousID := nodes[len(nodes)-4].ID
	edges = append(edges,
		MissionFlowEdge{
			ID:     previousID + "-to-check-understanding",
			Source: previousID,
			Target: "check-understanding",
			Label:  "Check",
		},
		MissionFlowEdge{
			ID:     "check-understanding-to-adaptive-feedback",
			Source: "check-understanding",
			Target: "adaptive-feedback",
			Label:  "Needs support",
		},
		MissionFlowEdge{
			ID:     "adaptive-feedback-to-complete",
			Source: "adaptive-feedback",
			Target: "complete",
			Label:  "Ready",
		},
		MissionFlowEdge{
			ID:     "check-understanding-to-complete",
			Source: "check-understanding",
			Target: "complete",
			Label:  "Understands",
		},
	)

	return MissionFlow{Nodes: nodes, Edges: edges}
}

func guardrailMissionFlow() MissionFlow {
	return MissionFlow{
		Nodes: []MissionFlowNode{
			{
				ID:          "start",
				Kind:        "start",
				Label:       "Teacher requests preview",
				Description: "Mia receives the class context, prompt, and attached files.",
				Position:    MissionFlowPosition{X: 0, Y: 0},
			},
			{
				ID:          "guardrail-check",
				Kind:        "guardrail",
				Label:       "Education guardrail",
				Description: "The source must be age-appropriate classroom learning material.",
				Position:    MissionFlowPosition{X: 0, Y: 130},
			},
			{
				ID:          "replace-material",
				Kind:        "setup",
				Label:       "Replace source",
				Description: "Attach a readable grade-level worksheet or paste a classroom learning prompt.",
				Position:    MissionFlowPosition{X: 0, Y: 260},
			},
			{
				ID:          "regenerate",
				Kind:        "complete",
				Label:       "Regenerate safely",
				Description: "Mia can create a student preview once the material is educational.",
				Position:    MissionFlowPosition{X: 0, Y: 390},
			},
		},
		Edges: []MissionFlowEdge{
			{ID: "start-to-guardrail-check", Source: "start", Target: "guardrail-check", Label: "Review"},
			{ID: "guardrail-check-to-replace-material", Source: "guardrail-check", Target: "replace-material", Label: "Blocked"},
			{ID: "replace-material-to-regenerate", Source: "replace-material", Target: "regenerate", Label: "Try again"},
		},
	}
}

func normalizeFlowKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "start", "setup", "activity", "question", "decision", "feedback", "complete", "guardrail":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "activity"
	}
}

func normalizeFlowID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	builder := strings.Builder{}
	lastWasDash := false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastWasDash = false
		case char == '-' || char == '_' || char == ' ':
			if !lastWasDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastWasDash = true
			}
		}
	}

	return strings.Trim(builder.String(), "-")
}

func clampFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}

	return value
}
