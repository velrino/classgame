package students

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/auth"
	"classgame-api/internal/transport/httpx"
)

type StudentClassReader interface {
	GetStudentClass(context.Context, auth.Identity) (StudentClassView, error)
}
type StudentClassView struct {
	StudentSessionClass
	Status          string `json:"status"`
	ActiveMissionID string `json:"active_mission_id"`
}

func (service *Service) GetStudentClass(ctx context.Context, identity auth.Identity) (StudentClassView, error) {
	if identity.AuthMethod != auth.AuthMethodAnonymous {
		return StudentClassView{}, ErrStudentRequiresAnonymousAuth
	}
	if service.sessions == nil {
		return StudentClassView{}, errors.New("student repository is not configured")
	}
	classID, classCode, err := service.sessions.GetStudentClassMembership(ctx, identity.ProviderUserID)
	if err != nil {
		return StudentClassView{}, err
	}
	class, err := service.classByCode(ctx, classCode)
	if err != nil {
		return StudentClassView{}, err
	}
	if class.ClassID != classID {
		return StudentClassView{}, ErrClassCodeNotFound
	}
	return StudentClassView{StudentSessionClass: studentSessionClassFromDocument(class), Status: class.Status, ActiveMissionID: class.ActiveMissionID}, nil
}
func (handler *Handler) GetStudentClass(ctx *echo.Context) error {
	if handler.authenticator == nil {
		return httpx.JSONError(ctx, http.StatusServiceUnavailable, "not_configured", "Student authentication is not configured.")
	}
	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.studentSessionTimeout)
	defer cancel()
	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	if handler.studentClassReader == nil {
		return httpx.JSONError(ctx, http.StatusServiceUnavailable, "not_configured", "Student class is not configured.")
	}
	view, err := handler.studentClassReader.GetStudentClass(requestContext, identity)
	if errors.Is(err, ErrClassCodeNotFound) {
		return httpx.JSONError(ctx, http.StatusNotFound, "class_not_found", "Active class code not found.")
	}
	if err != nil {
		return handler.studentLearningError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"class": view})
}
