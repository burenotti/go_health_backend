package api

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/app/authapp"
	inviteservice "github.com/burenotti/go_health_backend/internal/app/invite"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/invite"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (s *Server) MountInvites() {
	loginRequired := LoginRequired(s.authService.Authorizer)
	s.handler.POST("/groups/:group_id/invites", s.CreateInvite, loginRequired)
	s.handler.POST("/invites/accept", s.AcceptInvite, loginRequired)

}

func (s *Server) getInviteUoW() *unitofwork.UnitOfWork[*inviteservice.AtomicContext] {
	return unitofwork.New[*inviteservice.AtomicContext](
		s.db,
		inviteservice.NewAtomicContext,
		s.msgBus,
		s.logger,
	)
}

type CreateInviteRequest struct {
	GroupID string `param:"group_id"`
}

type CreateInviteResponse struct {
	GroupID    string    `json:"group_id"`
	InviteID   string    `json:"invite_id"`
	Secret     string    `json:"secret"`
	ValidUntil time.Time `json:"valid_until"`
}

func (s *Server) CreateInvite(c echo.Context) error {
	var req CreateInviteRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}
	uow := s.getInviteUoW()
	ctx := c.Request().Context()
	groupId := invite.GroupID(req.GroupID)

	inv, err := s.inviteService.CreateInvite(ctx, uow, groupId)

	if err != nil {

		if errors.Is(err, invite.ErrInviteExists) {
			return JsonError(c, http.StatusBadRequest, err)
		}

		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusCreated, CreateInviteResponse{
		GroupID:    string(inv.GroupID),
		InviteID:   string(inv.InviteID),
		Secret:     inv.Secret,
		ValidUntil: inv.ValidUntil,
	})
}

type AcceptInviteRequest struct {
	Secret string `json:"secret"`
}

func (s *Server) AcceptInvite(c echo.Context) error {
	var req AcceptInviteRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}
	uow := s.getInviteUoW()
	ctx := c.Request().Context()
	user := c.Get(KeyCurrentUser).(*authapp.AccessTokenData)

	err := s.inviteService.AcceptInvite(ctx, uow, invite.TraineeID(user.UserID), req.Secret)

	if err != nil {
		if errors.Is(err, invite.ErrInviteExpired) {
			return JsonError(c, http.StatusBadRequest, err)
		}

		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusOK)
}
