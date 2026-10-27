package students

import (
	"context"
	"errors"
	"time"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/tenancy"
)

func (service *Service) UpsertStudent(
	ctx context.Context,
	identity auth.Identity,
	classCode string,
	studentName string,
) (UserDocument, error) {
	if identity.AuthMethod != auth.AuthMethodAnonymous {
		return UserDocument{}, ErrStudentRequiresAnonymousAuth
	}

	teacherClass, err := service.classByCode(ctx, classCode)
	if err != nil {
		return UserDocument{}, err
	}

	user, err := service.sessions.UpsertStudent(ctx, UpsertStudentInput{
		Identity: identity,
		Scope: tenancy.Scope{
			OrganizationID: teacherClass.OrganizationID,
			DistrictID:     teacherClass.DistrictID,
			SchoolID:       teacherClass.SchoolID,
		},
		ClassID:        teacherClass.ClassID,
		ClassCode:      teacherClass.ClassCode,
		StudentName:    normalizeStudentName(studentName),
		IncrementLogin: true,
		Timestamp:      time.Now(),
	})
	if err != nil {
		return UserDocument{}, err
	}
	user.SessionClass = studentSessionClassFromDocument(teacherClass)

	return user, nil
}

func (service *Service) ValidateStudentClassCode(
	ctx context.Context,
	classCode string,
) error {
	// The public identifier check must not reject an existing member. New
	// admission is enforced with membership inside the session transaction.
	class, err := service.classByCode(ctx, classCode)
	if err == nil && !class.Active {
		return ErrClassCodeNotFound
	}
	return err
}

func (service *Service) StartStudentMission(
	ctx context.Context,
	identity auth.Identity,
	classCode string,
	missionID string,
	studentName string,
) (UserDocument, error) {
	if identity.AuthMethod != auth.AuthMethodAnonymous {
		return UserDocument{}, ErrStudentRequiresAnonymousAuth
	}
	teacherClass, err := service.classByCode(ctx, classCode)
	if err != nil {
		return UserDocument{}, err
	}
	missionDocument, err := service.publishedMissionByID(
		ctx,
		teacherClass.ClassID,
		missionID,
	)
	if err != nil {
		return UserDocument{}, err
	}
	user, err := service.sessions.UpsertStudent(ctx, UpsertStudentInput{
		Identity: identity,
		Scope: tenancy.Scope{
			OrganizationID: teacherClass.OrganizationID,
			DistrictID:     teacherClass.DistrictID,
			SchoolID:       teacherClass.SchoolID,
		},
		ClassID:                  teacherClass.ClassID,
		ClassCode:                teacherClass.ClassCode,
		MissionID:                missionDocument.ID,
		MissionCode:              missionDocument.MissionCode,
		MissionPublishedRevision: missionDocument.PublishedRevision,
		MissionSubject:           missionDocument.Subject,
		StudentName:              normalizeStudentName(studentName),
		Timestamp:                time.Now(),
	})
	if err != nil {
		return UserDocument{}, err
	}
	sessionMission, err := service.missionAtPublishedRevision(
		ctx,
		missionDocument,
		user.Activity.CurrentGame.PublishedRevision,
	)
	if err != nil {
		return UserDocument{}, err
	}
	user.SessionClass = studentSessionClassFromDocument(teacherClass)
	user.SessionMission = studentSessionMissionFromDocument(sessionMission)

	return user, nil
}

func (service *Service) classByCode(
	ctx context.Context,
	classCode string,
) (classrooms.TeacherClassDocument, error) {
	if service.classReader == nil {
		return classrooms.TeacherClassDocument{}, errors.New("student class reader is not configured")
	}
	teacherClass, exists, err := service.classReader.Get(ctx, normalizeClassCode(classCode))
	if err != nil {
		return classrooms.TeacherClassDocument{}, err
	}
	if !exists || teacherClass.Status != "active" {
		return classrooms.TeacherClassDocument{}, ErrClassCodeNotFound
	}

	return teacherClass, nil
}

func (service *Service) UpdateStudentPresence(
	ctx context.Context,
	identity auth.Identity,
	missionID string,
	classCode string,
	subject string,
	stageID string,
	stageName string,
	status string,
) (UserDocument, error) {
	if identity.AuthMethod != auth.AuthMethodAnonymous {
		return UserDocument{}, ErrStudentRequiresAnonymousAuth
	}
	teacherClass, err := service.classByCode(ctx, classCode)
	if err != nil {
		return UserDocument{}, err
	}
	normalizedMissionID := normalizeText(missionID)
	if _, err := service.publishedMissionByID(ctx, teacherClass.ClassID, normalizedMissionID); err != nil {
		return UserDocument{}, err
	}

	return service.sessions.UpdateStudentPresence(ctx, UpdateStudentPresenceInput{
		Identity:  identity,
		MissionID: normalizedMissionID,
		Subject:   normalizeText(subject),
		StageID:   normalizeText(stageID),
		StageName: normalizeText(stageName),
		Status:    normalizeText(status),
		Timestamp: time.Now(),
	})
}
