package classrooms

import (
	"regexp"
	"testing"
)

func TestGenerateClassCodeUsesSixDigits(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9]{6}$`)

	for index := 0; index < 100; index++ {
		code, err := generateClassCode()
		if err != nil {
			t.Fatalf("generate class code: %v", err)
		}
		if !pattern.MatchString(code) {
			t.Fatalf("generated code %q does not contain six digits", code)
		}
	}
}

func TestUpsertTeacherClassRequestAllowsGeneratedClassID(t *testing.T) {
	request := UpsertTeacherClassRequest{
		ClassName:  "Algebra Lab",
		Subject:    "Mathematics",
		GradeLevel: "7th grade",
	}

	if err := request.Validate(); err != nil {
		t.Fatalf("validate request without class id: %v", err)
	}
}

func TestUpsertTeacherClassRequestRejectsInvalidExistingClassID(t *testing.T) {
	request := UpsertTeacherClassRequest{
		ClassID:   "code with spaces",
		ClassName: "Algebra Lab",
		Subject:   "Mathematics",
	}

	if err := request.Validate(); err == nil {
		t.Fatal("expected invalid existing class id to fail validation")
	}
}
