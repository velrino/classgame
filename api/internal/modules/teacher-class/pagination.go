package teacherclass

import (
	"context"
	"fmt"

	"classgame-api/internal/pagination"
	sharedauth "classgame-api/internal/shared/auth"
	"cloud.google.com/go/firestore"
)

func (service *Service) ListPage(ctx context.Context, identity sharedauth.Identity, page pagination.Params) ([]TeacherClassDocument, string, error) {
	teacher, err := service.teacher(identity)
	if err != nil {
		return nil, "", err
	}
	return service.repository.ListPageByTeacherID(ctx, teacher.ProviderUserID, page)
}

func (repository *Repository) ListPageByTeacherID(ctx context.Context, teacherID string, page pagination.Params) ([]TeacherClassDocument, string, error) {
	query := repository.client.Collection(CollectionName).Where("teacher_id", "==", teacherID).
		OrderBy(firestore.DocumentID, firestore.Asc).Limit(page.Limit + 1)
	if page.After != "" {
		query = query.StartAfter(page.After)
	}
	documents, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, "", fmt.Errorf("list teacher class page: %w", err)
	}
	next := ""
	if len(documents) > page.Limit {
		documents = documents[:page.Limit]
		next = pagination.Cursor(documents[len(documents)-1].Ref.ID)
	}
	classes := make([]TeacherClassDocument, 0, len(documents))
	for _, snapshot := range documents {
		var document TeacherClassDocument
		if err := snapshot.DataTo(&document); err != nil {
			return nil, "", fmt.Errorf("decode teacher class page: %w", err)
		}
		document.ID = snapshot.Ref.ID
		hydrateTeacherClassTenantScope(&document)
		classes = append(classes, document)
	}
	return classes, next, nil
}
