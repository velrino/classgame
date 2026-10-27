package students

import (
	"context"
	"errors"

	"classgame-api/internal/modules/missions"
)

func (service *Service) publishedMissionByID(
	ctx context.Context,
	classID string,
	missionID string,
) (missions.MissionDocument, error) {
	if service.missionReader == nil {
		return missions.MissionDocument{}, errors.New("student mission reader is not configured")
	}
	normalizedClassID := normalizeClassCode(classID)
	normalizedMissionID := normalizeText(missionID)
	missionDocument, exists, err := service.missionReader.Get(
		ctx,
		normalizedClassID,
		normalizedMissionID,
	)
	if err != nil {
		return missions.MissionDocument{}, err
	}
	if !exists ||
		missionDocument.ID != normalizedMissionID ||
		missionDocument.ClassID != normalizedClassID {
		return missions.MissionDocument{}, ErrPublishedMissionNotFound
	}
	published, available := missions.PublishedView(missionDocument)
	if !available || published.AudienceType == missions.AudienceSelected {
		return missions.MissionDocument{}, ErrPublishedMissionNotFound
	}

	return published, nil
}

func (service *Service) publishedMissionByCode(
	ctx context.Context,
	missionCode string,
) (missions.MissionDocument, error) {
	if service.missionReader == nil {
		return missions.MissionDocument{}, errors.New("student mission reader is not configured")
	}

	missionDocument, exists, err := service.missionReader.GetByCode(ctx, normalizeMissionCode(missionCode))
	if err != nil {
		return missions.MissionDocument{}, err
	}
	if !exists || missionDocument.Status != missions.StatusPublished {
		return missions.MissionDocument{}, ErrMissionCodeNotFound
	}

	return missionDocument, nil
}

func (service *Service) missionAtPublishedRevision(
	ctx context.Context,
	current missions.MissionDocument,
	revision int64,
) (missions.MissionDocument, error) {
	if revision <= 0 || current.PublishedRevision == revision {
		return current, nil
	}

	revisionReader := service.revisionReader
	if revisionReader == nil {
		return missions.MissionDocument{}, ErrMissionRevisionNotFound
	}
	published, exists, err := revisionReader.GetPublishedRevision(
		ctx,
		current.ClassID,
		current.ID,
		revision,
	)
	if err != nil {
		return missions.MissionDocument{}, err
	}
	if !exists {
		return missions.MissionDocument{}, ErrMissionRevisionNotFound
	}
	view, valid := missions.ViewAtPublishedRevision(current, published)
	if !valid {
		return missions.MissionDocument{}, ErrMissionRevisionNotFound
	}

	return view, nil
}

func (service *Service) missionForAttempt(
	ctx context.Context,
	attempt StudentMissionAttemptDocument,
) (missions.MissionDocument, error) {
	if revisionReader := service.revisionReader; revisionReader != nil && attempt.PublishedRevision > 0 {
		published, exists, err := revisionReader.GetPublishedRevision(
			ctx,
			attempt.ClassID,
			attempt.MissionID,
			attempt.PublishedRevision,
		)
		if err != nil {
			return missions.MissionDocument{}, err
		}
		if !exists {
			return missions.MissionDocument{}, ErrMissionRevisionNotFound
		}
		view, valid := missions.ViewAtPublishedRevision(missions.MissionDocument{
			ID:          attempt.MissionID,
			ClassID:     attempt.ClassID,
			MissionCode: attempt.MissionCode,
		}, published)
		if !valid {
			return missions.MissionDocument{}, ErrMissionRevisionNotFound
		}

		return view, nil
	}

	current, err := service.publishedMissionByID(ctx, attempt.ClassID, attempt.MissionID)
	if err != nil {
		return missions.MissionDocument{}, err
	}

	return service.missionAtPublishedRevision(ctx, current, attempt.PublishedRevision)
}
