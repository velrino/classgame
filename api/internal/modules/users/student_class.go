package users

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	sharedauth "classgame-api/internal/shared/auth"
	"github.com/labstack/echo/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StudentClassReader interface {
	GetStudentClass(context.Context, sharedauth.Identity) (StudentClassView, error)
}
type StudentClassView struct {
	StudentSessionClass
	Status          string `json:"status"`
	ActiveMissionID string `json:"active_mission_id"`
}

func (service *Service) GetStudentClass(ctx context.Context, identity sharedauth.Identity) (StudentClassView, error) {
	if identity.AuthMethod != sharedauth.AuthMethodAnonymous {
		return StudentClassView{}, ErrStudentRequiresAnonymousAuth
	}
	if service.repository == nil {
		return StudentClassView{}, errors.New("student repository is not configured")
	}
	snapshot, err := service.repository.client.Collection(usersCollectionName).Doc(identity.ProviderUserID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return StudentClassView{}, ErrStudentNotFound
	}
	if err != nil {
		return StudentClassView{}, fmt.Errorf("read student class session: %w", err)
	}
	var user UserDocument
	if err := snapshot.DataTo(&user); err != nil {
		return StudentClassView{}, err
	}
	class, err := service.classByCode(ctx, user.Student.ClassCode)
	if err != nil {
		return StudentClassView{}, err
	}
	if class.ClassID != user.Student.ClassID {
		return StudentClassView{}, ErrClassCodeNotFound
	}
	return StudentClassView{StudentSessionClass: studentSessionClassFromDocument(class), Status: class.Status, ActiveMissionID: class.ActiveMissionID}, nil
}
func (handler *Handler) GetStudentClass(ctx *echo.Context) error {
	if handler.authenticator == nil {
		return jsonError(ctx, http.StatusServiceUnavailable, "not_configured", "Student authentication is not configured.")
	}
	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentSessionTimeout)
	defer cancel()
	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	if handler.studentClassReader == nil {
		return jsonError(ctx, http.StatusServiceUnavailable, "not_configured", "Student class is not configured.")
	}
	view, err := handler.studentClassReader.GetStudentClass(requestContext, identity)
	if errors.Is(err, ErrClassCodeNotFound) {
		return jsonError(ctx, http.StatusNotFound, "class_not_found", "Active class code not found.")
	}
	if err != nil {
		return handler.studentLearningError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"class": view})
}
