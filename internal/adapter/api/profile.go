package api

import (
	"errors"
	"github.com/burenotti/go_health_backend/internal/app/authapp"
	profileservice "github.com/burenotti/go_health_backend/internal/app/profile"
	"github.com/burenotti/go_health_backend/internal/app/unitofwork"
	"github.com/burenotti/go_health_backend/internal/domain/profile"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (s *Server) MountProfile() {
	s.handler.POST("/trainees/:user_id", s.CreateTrainee)
	s.handler.GET("/trainees/:user_id", s.GetTraineeByID)

	s.handler.POST("/coaches/:user_id", s.CreateCoach)
	s.handler.GET("/coaches/:user_id", s.GetCoachByID)

	s.handler.GET("/profiles/me", s.GetMyProfile, LoginRequired(s.authService.Authorizer))
}

func (s *Server) getProfileUoW() *unitofwork.UnitOfWork[*profileservice.AtomicContext] {
	return unitofwork.New[*profileservice.AtomicContext](
		s.db,
		profileservice.NewAtomicContext,
		s.msgBus,
		s.logger,
	)
}

type CreateTraineeRequest struct {
	UserID    string     `param:"user_id"`
	FirstName string     `json:"first_name,omitempty"`
	LastName  string     `json:"last_name,omitempty"`
	BirthDate *time.Time `json:"birth_date,omitempty"`
}

func (s *Server) CreateTrainee(c echo.Context) error {
	var req CreateTraineeRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	uow := s.getProfileUoW()
	ctx := c.Request().Context()
	_, err := s.profileService.CreateTrainee(ctx, req.UserID, req.FirstName, req.LastName, req.BirthDate, uow)
	if err != nil {
		if errors.Is(err, profile.ErrProfileExists) {
			return JsonError(c, http.StatusNotFound, "profile already exists")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusNoContent)
}

type CreateCoachRequest struct {
	UserID          string     `param:"user_id"`
	FirstName       string     `json:"first_name,omitempty"`
	LastName        string     `json:"last_name,omitempty"`
	BirthDate       *time.Time `json:"birth_date,omitempty"`
	YearsExperience int        `json:"years_experience,omitempty"`
	Bio             string     `json:"bio,omitempty"`
}

func (s *Server) CreateCoach(c echo.Context) error {
	var req CreateCoachRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	uow := s.getProfileUoW()
	ctx := c.Request().Context()

	_, err := s.profileService.CreateCoach(
		ctx,
		req.UserID,
		req.FirstName,
		req.LastName,
		req.BirthDate,
		req.YearsExperience,
		req.Bio,
		uow,
	)

	if err != nil {
		if errors.Is(err, profile.ErrProfileExists) {
			return JsonError(c, http.StatusNotFound, "profile already exists")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusNoContent)
}

type GetCoachByIDRequest struct {
	UserID string `param:"user_id"`
}

type GetCoachByIDResponse struct {
	UserID          string     `json:"user_id"`
	Type            string     `json:"type"`
	FirstName       string     `json:"first_name,omitempty"`
	LastName        string     `json:"last_name,omitempty"`
	BirthDate       *time.Time `json:"birth_date,omitempty"`
	YearsExperience int        `json:"years_experience,omitempty"`
	Bio             string     `json:"bio,omitempty"`
}

func (s *Server) GetCoachByID(c echo.Context) error {
	var req GetCoachByIDRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	uow := s.getProfileUoW()

	coach, err := s.profileService.GetCoachByID(c.Request().Context(), req.UserID, uow)
	if err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			return JsonError(c, http.StatusNotFound, "profile not found")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, GetCoachByIDResponse{
		UserID:          coach.UserID,
		Type:            coach.Type(),
		FirstName:       coach.FirstName,
		LastName:        coach.LastName,
		BirthDate:       coach.BirthDate,
		YearsExperience: coach.YearsExperience,
		Bio:             coach.Bio,
	})

}

type GetTraineeByIDRequest struct {
	UserID string `param:"user_id"`
}

type GetTraineeByIDResponse struct {
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"`
	FirstName string     `json:"first_name,omitempty"`
	LastName  string     `json:"last_name,omitempty"`
	BirthDate *time.Time `json:"birth_date,omitempty"`
}

func (s *Server) GetTraineeByID(c echo.Context) error {
	var req GetTraineeByIDRequest
	if err := s.bind(c, &req); err != nil {
		return JsonError(c, http.StatusBadRequest, err)
	}

	uow := s.getProfileUoW()

	t, err := s.profileService.GetTraineeByID(c.Request().Context(), req.UserID, uow)
	if err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			return JsonError(c, http.StatusNotFound, "profile not found")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, GetTraineeByIDResponse{
		UserID:    t.UserID,
		Type:      t.Type(),
		FirstName: t.FirstName,
		LastName:  t.LastName,
		BirthDate: t.BirthDate,
	})
}

func (s *Server) GetMyProfile(c echo.Context) error {
	user := c.Get(KeyCurrentUser).(*authapp.AccessTokenData)
	uow := s.getProfileUoW()

	p, err := s.profileService.GetProfileByID(c.Request().Context(), user.UserID, uow)
	if err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			return JsonError(c, http.StatusNotFound, "profile not found")
		}
		return JsonError(c, http.StatusInternalServerError, err)
	}

	switch v := p.(type) {
	case *profile.Coach:
		return c.JSON(http.StatusOK, GetCoachByIDResponse{
			UserID:          v.UserID,
			Type:            v.Type(),
			FirstName:       v.FirstName,
			LastName:        v.LastName,
			BirthDate:       v.BirthDate,
			YearsExperience: v.YearsExperience,
			Bio:             v.Bio,
		})
	case *profile.Trainee:
		return c.JSON(http.StatusOK, GetTraineeByIDResponse{
			UserID:    v.UserID,
			Type:      v.Type(),
			FirstName: v.FirstName,
			LastName:  v.LastName,
			BirthDate: v.BirthDate,
		})
	default:
		panic("unknown profile type")
	}
}
