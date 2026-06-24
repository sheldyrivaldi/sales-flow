package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	u, access, refresh, err := h.svc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return httperr.Write(c, err)
	}

	return c.JSON(http.StatusOK, dto.LoginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         dto.ToUserResponse(*u),
	})
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req dto.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	access, refresh, err := h.svc.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return httperr.Write(c, err)
	}

	return c.JSON(http.StatusOK, dto.RefreshResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	})
}

func (h *AuthHandler) Me(c echo.Context) error {
	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
	}

	u, err := h.svc.Me(c.Request().Context(), user.ID)
	if err != nil {
		return httperr.Write(c, err)
	}

	return c.JSON(http.StatusOK, dto.ToUserResponse(*u))
}
