package missions

import (
	"regexp"
	"testing"
)

func TestBuildMissionDocumentKeepsClassIDAndSharingCodeSeparate(t *testing.T) {
	mission := buildMissionDocument("mission-1", CreateMissionInput{
		ClassID:   "internal-class-id",
		ClassCode: "ABC234",
	})

	if mission.ClassID != "internal-class-id" {
		t.Fatalf("expected internal class id, got %q", mission.ClassID)
	}
	if mission.ClassCode != "ABC234" {
		t.Fatalf("expected sharing code, got %q", mission.ClassCode)
	}
}

func TestGenerateMissionCodeUsesNumericCharacters(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9]{6}$`)

	for index := 0; index < 100; index++ {
		code, err := generateMissionCode()
		if err != nil {
			t.Fatalf("generate mission code: %v", err)
		}
		if !pattern.MatchString(code) {
			t.Fatalf("generated mission code %q is not numeric", code)
		}
	}
}
