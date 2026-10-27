package teacherclass

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	sharedauth "classgame-api/internal/shared/auth"
)

const (
	classCodeAlphabet       = "0123456789"
	classCodeLength         = 6
	classCodeCreateAttempts = 12
)

var (
	ErrTeacherRequiresGoogleAuth = errors.New("teacher classes require google auth")
	ErrClassCodeGenerationFailed = errors.New("could not generate a unique teacher class code")
)

type Service struct {
	repository ClassroomStore
}

func NewService(repository ClassroomStore) *Service {
	return &Service{
		repository: repository,
	}
}

func (service *Service) Get(
	ctx context.Context,
	identity sharedauth.Identity,
	classID string,
) (TeacherClassDocument, bool, error) {
	teacher, err := service.teacher(identity)
	if err != nil {
		return TeacherClassDocument{}, false, err
	}

	teacherClass, exists, err := service.repository.Get(ctx, normalizeClassID(classID))
	if err != nil || !exists {
		return teacherClass, exists, err
	}
	if teacherClass.TeacherID != teacher.ProviderUserID {
		return TeacherClassDocument{}, true, ErrClassOwnedByAnotherTeacher
	}

	return teacherClass, true, nil
}

func (service *Service) List(
	ctx context.Context,
	identity sharedauth.Identity,
) ([]TeacherClassDocument, error) {
	teacher, err := service.teacher(identity)
	if err != nil {
		return nil, err
	}

	return service.repository.ListByTeacherID(ctx, teacher.ProviderUserID)
}

func (service *Service) Upsert(
	ctx context.Context,
	identity sharedauth.Identity,
	input UpsertTeacherClassInput,
) (TeacherClassDocument, error) {
	teacher, err := service.teacher(identity)
	if err != nil {
		return TeacherClassDocument{}, err
	}

	input.Teacher = teacher
	input.ClassID = normalizeClassID(input.ClassID)
	input.ClassName = normalizeText(input.ClassName)
	input.Subject = normalizeText(input.Subject)
	input.GradeLevel = normalizeText(input.GradeLevel)
	input.Timestamp = time.Now()

	if input.ClassID == "" {
		return service.createWithGeneratedCode(ctx, input)
	}

	if input.ClassName == "" {
		input.ClassName = input.ClassID
	}
	if input.Subject == "" {
		input.Subject = "Mathematics"
	}
	if input.GradeLevel == "" {
		input.GradeLevel = "7th grade"
	}

	return service.repository.Upsert(ctx, input)
}

func (service *Service) createWithGeneratedCode(
	ctx context.Context,
	input UpsertTeacherClassInput,
) (TeacherClassDocument, error) {
	useGeneratedName := input.ClassName == ""

	for attempt := 0; attempt < classCodeCreateAttempts; attempt++ {
		classCode, err := generateClassCode()
		if err != nil {
			return TeacherClassDocument{}, fmt.Errorf("generate teacher class code: %w", err)
		}

		input.ClassID = classCode
		if useGeneratedName {
			input.ClassName = classCode
		}

		teacherClass, err := service.repository.Create(ctx, input)
		if errors.Is(err, ErrClassCodeUnavailable) {
			continue
		}
		return teacherClass, err
	}

	return TeacherClassDocument{}, ErrClassCodeGenerationFailed
}

func (service *Service) SetActiveMission(
	ctx context.Context,
	identity sharedauth.Identity,
	classID string,
	missionID string,
) error {
	teacher, err := service.teacher(identity)
	if err != nil {
		return err
	}

	return service.repository.SetActiveMission(
		ctx,
		normalizeClassID(classID),
		teacher.ProviderUserID,
		missionID,
		time.Now(),
	)
}

func (service *Service) TeacherFromIdentity(identity sharedauth.Identity) (TeacherSummary, error) {
	return service.teacher(identity)
}

func (service *Service) teacher(identity sharedauth.Identity) (TeacherSummary, error) {
	if !sharedauth.IsTeacherAuthMethod(identity.AuthMethod) {
		return TeacherSummary{}, ErrTeacherRequiresGoogleAuth
	}

	displayName := strings.TrimSpace(identity.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(identity.Email)
	}

	return TeacherSummary{
		ProviderUserID: identity.ProviderUserID,
		DisplayName:    displayName,
		Email:          strings.TrimSpace(identity.Email),
		PhotoURL:       strings.TrimSpace(identity.PhotoURL),
	}, nil
}

func normalizeClassID(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func generateClassCode() (string, error) {
	code := make([]byte, classCodeLength)
	alphabetLength := big.NewInt(int64(len(classCodeAlphabet)))

	for index := range code {
		characterIndex, err := rand.Int(rand.Reader, alphabetLength)
		if err != nil {
			return "", err
		}
		code[index] = classCodeAlphabet[characterIndex.Int64()]
	}

	return string(code), nil
}
