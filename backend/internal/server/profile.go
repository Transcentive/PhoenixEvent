package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Transcentive/PhoenixEvent/backend/internal/accounts"
)

// The profile (spec 8B): A8 for its owner, A9 for anyone else logged in to
// the organization.

func account(c *gin.Context) *accounts.Account {
	return c.MustGet(ctxAccount).(*accounts.Account)
}

func (h *handlers) profile(c *gin.Context) {
	p, err := h.accounts.GetProfile(c.Request.Context(), organization(c), account(c).ID)
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *handlers) updateProfile(c *gin.Context) {
	var body accounts.ProfileChange
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	p, err := h.accounts.UpdateProfile(c.Request.Context(), organization(c), account(c).ID, body)
	if h.fieldErrors(c, err) {
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *handlers) updateVisibility(c *gin.Context) {
	var body accounts.VisibilityChange
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	p, err := h.accounts.UpdateVisibility(c.Request.Context(), organization(c), account(c).ID, body)
	if h.fieldErrors(c, err) {
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

type changeEmailBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *handlers) changeEmail(c *gin.Context) {
	var body changeEmailBody
	_ = c.ShouldBindJSON(&body)
	err := h.accounts.ChangeEmail(c.Request.Context(), organization(c), account(c).ID, body.Email, body.Password)
	if h.fieldErrors(c, err) {
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	h.profile(c)
}

// person is A9. The person is looked up within the session's organization
// only, so another organization's account is never found.
func (h *handlers) person(c *gin.Context) {
	p, err := h.accounts.Shared(c.Request.Context(), organization(c), account(c).ID, c.Param("id"))
	if errors.Is(err, accounts.ErrNotFound) {
		abort(c, http.StatusNotFound, "not_found", "There is no such person in this organization.")
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}
