package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Transcentive/PhoenixEvent/backend/internal/accounts"
)

// Organization roles (7B): O8, Organization officers, and D5, Change roles.

func (h *handlers) officers(c *gin.Context) {
	list, err := h.accounts.Officers(c.Request.Context(), organization(c))
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"officers": list})
}

func (h *handlers) personRoles(c *gin.Context) {
	p, err := h.accounts.RolesFor(c.Request.Context(), organization(c), account(c).ID, c.Param("id"))
	if h.refused(c, err) {
		return
	}
	c.JSON(http.StatusOK, p)
}

type rolesBody struct {
	Roles []string `json:"roles"`
}

func (h *handlers) setRoles(c *gin.Context) {
	var body rolesBody
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	err := h.accounts.SetRoles(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), body.Roles)
	if h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	h.personRoles(c)
}

// refused answers a missing person and a missing role alike (E3), so the
// response says nothing about what lies outside the person's access.
func (h *handlers) refused(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, accounts.ErrNotFound), errors.Is(err, accounts.ErrForbidden):
		abort(c, http.StatusNotFound, "no_access", "This page is outside your access.")
	default:
		h.internal(c, err)
	}
	return true
}
