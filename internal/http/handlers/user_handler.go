package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) List(c echo.Context) error {
	f := domain.UserFilter{}

	if r := c.QueryParam("role"); r != "" {
		role := domain.Role(r)
		f.Role = &role
	}
	switch c.QueryParam("active") {
	case "true":
		v := true
		f.Active = &v
	case "false":
		v := false
		f.Active = &v
	}
	f.Search = c.QueryParam("search")

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))

	users, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	items := make([]dto.UserResponse, len(users))
	for i, u := range users {
		items[i] = dto.ToUserResponse(u)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	return c.JSON(http.StatusOK, dto.UserListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *UserHandler) Create(c echo.Context) error {
	var req dto.UserCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	u, err := h.svc.Create(c.Request().Context(), req.Email, req.Name, domain.Role(req.Role), req.Password)
	if err != nil {
		return httperr.Write(c, err)
	}

	return c.JSON(http.StatusCreated, dto.ToUserResponse(*u))
}

func (h *UserHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req dto.UserUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	u, err := h.svc.Update(c.Request().Context(), id, req.Name, req.Role, req.Active)
	if err != nil {
		return httperr.Write(c, err)
	}

	return c.JSON(http.StatusOK, dto.ToUserResponse(*u))
}

func (h *UserHandler) ResetPassword(c echo.Context) error {
	id := c.Param("id")

	var req dto.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	plaintext, err := h.svc.ResetPassword(c.Request().Context(), id, req.Password)
	if err != nil {
		return httperr.Write(c, err)
	}

	return c.JSON(http.StatusOK, dto.ResetPasswordResponse{Password: plaintext})
}
