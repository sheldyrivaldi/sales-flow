package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

type KeywordHandler struct {
	svc *service.KeywordService
}

func NewKeywordHandler(svc *service.KeywordService) *KeywordHandler {
	return &KeywordHandler{svc: svc}
}

// Generate handles POST /api/profile/keywords/generate — returns a draft
// keyword set (not persisted); caller reviews/edits then saves via PUT /api/profile.
func (h *KeywordHandler) Generate(c echo.Context) error {
	var req dto.KeywordGenerateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	language := ""
	if req.Language != nil {
		language = *req.Language
	}

	resp, err := h.svc.Generate(c.Request().Context(), req.ServiceCategories, language)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, resp)
}
