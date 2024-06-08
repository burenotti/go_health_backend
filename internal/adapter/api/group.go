package api

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/app/authapp"
	groupservice "github.com/burenotti/go_health_backend/internal/app/group"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/group"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"net/http"
)

func (s *Server) MountGroups() {
	loginRequired := LoginRequired(s.authService.Authorizer)
	groupsGroup := s.handler.Group("/groups", loginRequired)

	groupsGroup.GET("/list", s.GetGroupsList)
	groupsGroup.POST("/:group_id", s.CreateGroup)
	groupsGroup.GET("/:group_id", s.GetGroup)
	groupsGroup.GET("/:group_id/members", s.GetGroupMembers)
}

func (s *Server) getGroupUoW() *unitofwork.UnitOfWork[*groupservice.AtomicContext] {
	return unitofwork.New[*groupservice.AtomicContext](
		s.db,
		groupservice.NewAtomicContext,
		s.msgBus,
		s.logger,
	)
}

type CreateGroupRequest struct {
	GroupID     string `param:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) CreateGroup(c echo.Context) error {
	var req CreateGroupRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}
	user := c.Get(KeyCurrentUser).(*authapp.AccessTokenData)
	uow := s.getGroupUoW()
	ctx := c.Request().Context()
	groupId := group.GroupID(req.GroupID)
	coachId := group.CoachID(user.UserID)
	err := s.groupService.CreateGroup(ctx, uow, groupId, coachId, req.Name, req.Description)

	if err != nil {

		if errors.Is(err, group.ErrGroupExists) {
			return JsonError(c, http.StatusBadRequest, err)
		}

		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusCreated)
}

type GetGroupRequest struct {
	GroupID string `param:"group_id"`
}

type GetGroupResponse struct {
	GroupID     string `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) GetGroup(c echo.Context) error {
	var req GetGroupRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	uow := s.getGroupUoW()

	g, err := s.groupService.GetByID(c.Request().Context(), uow, group.GroupID(req.GroupID))

	if err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			return JsonError(c, http.StatusNotFound, err)
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, GetGroupResponse{
		GroupID:     string(g.GroupID),
		Name:        g.Name,
		Description: g.Description,
	})
}

type GetGroupMembersRequest struct {
	GroupID string `param:"group_id"`
	Limit   int    `query:"limit"`
	Offset  int    `query:"offset"`
}

type Member struct {
	TraineeID string `json:"trainee_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type GetMembersResponse struct {
	Members []Member `json:"members"`
}

func (s *Server) GetGroupMembers(c echo.Context) error {
	var req GetGroupMembersRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	uow := s.getGroupUoW()

	ctx := c.Request().Context()
	groupId := group.GroupID(req.GroupID)
	members, err := s.groupService.GetMembers(ctx, uow, groupId, req.Limit, req.Offset)

	if err != nil {
		if errors.Is(err, group.ErrGroupNotFound) {
			return JsonError(c, http.StatusNotFound, err)
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}

	var resp GetMembersResponse

	for _, mem := range members {
		resp.Members = append(resp.Members, Member{
			TraineeID: string(mem.TraineeID),
			Email:     mem.Email,
			FirstName: mem.FirstName,
			LastName:  mem.LastName,
		})
	}

	return c.JSON(http.StatusOK, resp)
}

type Group struct {
	GroupID     string `json:"group_id"`
	CoachID     string `json:"coach_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GetGroupsListRequest struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

type GetGroupsListResponse struct {
	Groups []Group `json:"groups"`
}

func (s *Server) GetGroupsList(c echo.Context) error {
	var req GetGroupsListRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	user := c.Get(KeyCurrentUser).(*authapp.AccessTokenData)
	ctx := c.Request().Context()
	uow := s.getGroupUoW()
	list, err := s.groupService.GetUserGroups(ctx, uow, user.UserID, req.Limit, req.Offset)

	if err != nil {
		return JsonError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, GetGroupsListResponse{
		Groups: lo.Map(list, func(item *group.Group, index int) Group {
			return Group{
				GroupID:     string(item.GroupID),
				CoachID:     string(item.CoachID),
				Name:        item.Name,
				Description: item.Description,
			}
		}),
	})
}
