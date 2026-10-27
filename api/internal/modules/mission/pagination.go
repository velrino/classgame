package mission

import (
	"context"
	"fmt"
	"sort"

	"classgame-api/internal/pagination"
	sharedfirebase "classgame-api/internal/shared/google/firebase"
	"cloud.google.com/go/firestore"
)

func (service *Service) ListPage(ctx context.Context, identity sharedfirebase.Identity, classID string, page pagination.Params) ([]MissionDocument, string, error) {
	class, exists, err := service.teacherClassService.Get(ctx, identity, classID)
	if err != nil || !exists {
		return []MissionDocument{}, "", err
	}
	return service.repository.ListPage(ctx, class.ClassID, page)
}

func (repository *Repository) ListPage(ctx context.Context, classID string, page pagination.Params) ([]MissionDocument, string, error) {
	queries := []firestore.Query{
		repository.legacyClassMissionsQuery(classID).Query,
		repository.client.Collection(MissionsCollectionName).Where("class_id", "==", classID),
	}
	byID := make(map[string]*firestore.DocumentSnapshot)
	for _, query := range queries {
		query = query.OrderBy(firestore.DocumentID, firestore.Asc).Limit(page.Limit + 1)
		if page.After != "" {
			query = query.StartAfter(page.After)
		}
		documents, err := query.Documents(ctx).GetAll()
		if err != nil {
			return nil, "", fmt.Errorf("list mission page: %w", err)
		}
		for _, document := range documents {
			byID[document.Ref.ID] = document
		}
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	next := ""
	if len(ids) > page.Limit {
		ids = ids[:page.Limit]
		next = pagination.Cursor(ids[len(ids)-1])
	}
	missions := make([]MissionDocument, 0, len(ids))
	for _, id := range ids {
		mission, belongs, err := missionFromSnapshot(byID[id], classID)
		if err != nil {
			return nil, "", err
		}
		if belongs && mission.Status != StatusDeleted {
			missions = append(missions, mission)
		}
	}
	return missions, next, nil
}
