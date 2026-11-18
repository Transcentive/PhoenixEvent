package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Transcentive/PhoenixEvent/backend/internal/accounts"
)

// O3, Policies, and S1, Create event, with the events O1 lists (16B.6).

func (h *handlers) getPolicies(c *gin.Context) {
	p, err := h.accounts.GetPolicies(c.Request.Context(), organization(c), account(c).ID)
	if h.refused(c, err) {
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *handlers) setPolicies(c *gin.Context) {
	var body accounts.Policies
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	p, err := h.accounts.SetPolicies(c.Request.Context(), organization(c), account(c).ID, body)
	if h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *handlers) events(c *gin.Context) {
	list, err := h.accounts.Events(c.Request.Context(), organization(c), account(c).ID)
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": list})
}

func (h *handlers) createEvent(c *gin.Context) {
	var body accounts.NewEvent
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	e, err := h.accounts.CreateEvent(c.Request.Context(), organization(c), account(c).ID, body)
	if h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	c.JSON(http.StatusCreated, e)
}

func (h *handlers) getEvent(c *gin.Context) {
	e, err := h.accounts.GetEvent(c.Request.Context(), organization(c), account(c).ID, c.Param("id"))
	if h.refused(c, err) {
		return
	}
	c.JSON(http.StatusOK, e)
}

func (h *handlers) updateEvent(c *gin.Context) {
	var body accounts.NewEvent
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	e, err := h.accounts.UpdateEvent(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), body)
	if h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	c.JSON(http.StatusOK, e)
}

func (h *handlers) deleteEvent(c *gin.Context) {
	err := h.accounts.DeleteEvent(c.Request.Context(), organization(c), account(c).ID, c.Param("id"))
	if h.refused(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *handlers) board(c *gin.Context) {
	b, err := h.accounts.GetBoard(c.Request.Context(), organization(c), account(c).ID, c.Param("id"))
	if h.refused(c, err) {
		return
	}
	c.JSON(http.StatusOK, b)
}

type appointBody struct {
	Person   string `json:"person"`
	Position string `json:"position"`
}

func (h *handlers) appoint(c *gin.Context) {
	var body appointBody
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	err := h.accounts.Appoint(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), body.Person, body.Position)
	if h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	h.board(c)
}

func (h *handlers) recordChair(c *gin.Context) {
	var body appointBody
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	err := h.accounts.RecordChair(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), body.Person)
	if h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	h.board(c)
}

func (h *handlers) replaceChair(c *gin.Context) {
	var body accounts.ReplaceChairRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	err := h.accounts.ReplaceChair(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), body)
	if h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	h.board(c)
}

func (h *handlers) removeFromBoard(c *gin.Context) {
	err := h.accounts.RemoveFromBoard(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), c.Param("person"))
	if h.refused(c, err) {
		return
	}
	h.board(c)
}

// Removals from the event board (5D.3).

type removalBody struct {
	Person  string `json:"person"`
	Reason  string `json:"reason"`
	InFavor bool   `json:"in_favor"`
	Approve bool   `json:"approve"`
	Carries *bool  `json:"carries"`
}

func (h *handlers) removalAction(c *gin.Context, do func(body removalBody) error) {
	var body removalBody
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
			return
		}
	}
	if err := do(body); h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	h.board(c)
}

func (h *handlers) startRemoval(c *gin.Context) {
	h.removalAction(c, func(b removalBody) error {
		return h.accounts.StartRemoval(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), b.Person, b.Reason)
	})
}

func (h *handlers) editRemoval(c *gin.Context) {
	h.removalAction(c, func(b removalBody) error {
		return h.accounts.EditRemoval(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), c.Param("removal"), b.Reason)
	})
}

func (h *handlers) withdrawRemoval(c *gin.Context) {
	h.removalAction(c, func(removalBody) error {
		return h.accounts.WithdrawRemoval(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), c.Param("removal"))
	})
}

func (h *handlers) voteOnRemoval(c *gin.Context) {
	h.removalAction(c, func(b removalBody) error {
		return h.accounts.VoteOnRemoval(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), c.Param("removal"), b.InFavor)
	})
}

func (h *handlers) decideRemoval(c *gin.Context) {
	h.removalAction(c, func(b removalBody) error {
		return h.accounts.DecideRemoval(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), c.Param("removal"), b.Approve)
	})
}

func (h *handlers) closeVoting(c *gin.Context) {
	h.removalAction(c, func(b removalBody) error {
		return h.accounts.CloseVoting(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), c.Param("removal"), b.Carries)
	})
}

func (h *handlers) recordDeputy(c *gin.Context) {
	var body appointBody
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	err := h.accounts.RecordDeputy(c.Request.Context(), organization(c), account(c).ID, c.Param("id"), body.Person)
	if h.fieldErrors(c, err) || h.refused(c, err) {
		return
	}
	h.board(c)
}
