package students

import (
	"reflect"
	"testing"

	"classgame-api/internal/modules/ai"
	"classgame-api/internal/modules/missions"
)

func TestRetrievalTokensWordBoundaries(t *testing.T) {
	cases := []struct {
		name, input string
		want        []string
	}{
		{"ASCII quotes", `Explain "condensation"`, []string{"explain", "condensation"}},
		{"curly quotes", "Explain “condensation” ‘evaporation’", []string{"explain", "condensation", "evaporation"}},
		{"dash and punctuation", "Explain—condensation/evaporation!", []string{"explain", "condensation", "evaporation"}},
		{"case and duplicate words", "CONDENSATION condensation", []string{"condensation"}},
		{"existing byte minimum", "pa pá a abc", []string{"pá", "abc"}},
		{"combining mark retained", "cafe\u0301", []string{"cafe\u0301"}},
		{"no CJK segmentation", "水蒸气凝结", []string{"水蒸气凝结"}},
		{"empty punctuation", "“”—!?", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := make(map[string]struct{}, len(tc.want))
			for _, word := range tc.want {
				want[word] = struct{}{}
			}
			if got := retrievalTokens(tc.input); !reflect.DeepEqual(got, want) {
				t.Fatalf("tokens = %v, want %v", got, want)
			}
		})
	}
}

func TestRetrieveMissionKnowledgePunctuationAndBound(t *testing.T) {
	lesson := missions.MissionDocument{
		Subject: "Science", LearningGoal: "Understand physical processes",
		KnowledgeChunks: []ai.KnowledgeChunk{
			{ID: "condensation-source", Title: "Condensation", Content: "Condensation changes water vapor into liquid droplets."},
			{ID: "evaporation-source", Title: "Evaporation", Content: "Evaporation changes liquid water into water vapor."},
		},
	}
	cases := []struct {
		prompt string
		want   []string
	}{
		{"Explain condensation", []string{"condensation-source"}},
		{"Explain “condensation”", []string{"condensation-source"}},
		{"Explain—condensation", []string{"condensation-source"}},
		{"Explain ‘condensation’", []string{"condensation-source"}},
		{"Explain tectonics", nil},
		{"Explain evaporation", []string{"evaporation-source"}},
		{"condensation evaporation", []string{"condensation-source"}},
	}
	for _, tc := range cases {
		t.Run(tc.prompt, func(t *testing.T) {
			got := RetrieveMissionKnowledge(lesson, SaveStudentTutorTurnInput{Prompt: tc.prompt}, 1)
			if len(got) != len(tc.want) {
				t.Fatalf("sources = %v, want %v", got, tc.want)
			}
			for i, chunk := range got {
				if chunk.ID != tc.want[i] {
					t.Fatalf("source = %q, want %q", chunk.ID, tc.want[i])
				}
			}
		})
	}
}
