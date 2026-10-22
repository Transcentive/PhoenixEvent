package server

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Transcentive/PhoenixEvent/backend/internal/accounts"
	"github.com/Transcentive/PhoenixEvent/backend/internal/site"
	"github.com/Transcentive/PhoenixEvent/backend/internal/token"
)

const (
	// The sign-in cookie: for this one address only, so it carries no
	// Domain (4A.5), and sent only to the sign-in endpoints.
	sessionCookie     = "pe_session"
	sessionCookiePath = "/api/v1/auth"
	// The organization cookie, pe_org, for every organization's address
	// (4A.5). It says which organization the browser is in and grants
	// nothing.
	orgCookie = "pe_org"

	ctxOrganization = "organization"
	ctxAccount      = "account"
)

// withOrganization reads the organization from the address and refuses the
// request when the address is not one organization's.
func (h *handlers) withOrganization(c *gin.Context) {
	addr := h.resolve(c)
	if addr.Kind != site.Organization {
		abort(c, http.StatusNotFound, "unknown_address", "There is no organization at this address.")
		return
	}
	org, err := h.accounts.OrganizationBySlug(c.Request.Context(), addr.Slug)
	if errors.Is(err, accounts.ErrNotFound) {
		abort(c, http.StatusNotFound, "unknown_address", "There is no organization at this address.")
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	c.Set(ctxOrganization, org)
	c.Next()
}

// requireSession refuses a request without a valid access token for an open
// session in this organization.
func (h *handlers) requireSession(c *gin.Context) {
	raw, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !found {
		abort(c, http.StatusUnauthorized, "session_ended", accounts.ErrSessionEnded.Error())
		return
	}
	claims, err := h.tokens.Check(raw)
	if err != nil {
		abort(c, http.StatusUnauthorized, "session_ended", accounts.ErrSessionEnded.Error())
		return
	}
	acc, err := h.accounts.Authenticate(c.Request.Context(), organization(c), claims)
	if errors.Is(err, accounts.ErrSessionEnded) {
		abort(c, http.StatusUnauthorized, "session_ended", err.Error())
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	c.Set(ctxAccount, acc)
	c.Next()
}

// address tells the web client what kind of address it is at, and at an
// organization's address whether this browser is logged in to another
// organization (4A.5, rule 2).
func (h *handlers) address(c *gin.Context) {
	addr := h.resolve(c)
	switch addr.Kind {
	case site.General:
		c.JSON(http.StatusOK, gin.H{"kind": "general"})
		return
	case site.Unknown:
		abort(c, http.StatusNotFound, "unknown_address", "There is no organization at this address.")
		return
	}
	org, err := h.accounts.OrganizationBySlug(c.Request.Context(), addr.Slug)
	if errors.Is(err, accounts.ErrNotFound) {
		abort(c, http.StatusNotFound, "unknown_address", "There is no organization at this address.")
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	body := gin.H{"kind": "organization", "organization": org, "general_url": h.site.GeneralURL()}
	if other := h.otherOrganization(c, org); other != nil {
		body["logged_in_elsewhere"] = other
	}
	c.JSON(http.StatusOK, body)
}

// otherOrganization reads pe_org and, when it names a different organization,
// returns that organization and the address that logs the browser out of it.
func (h *handlers) otherOrganization(c *gin.Context, here *accounts.Organization) gin.H {
	slug, err := c.Cookie(orgCookie)
	if err != nil || slug == "" || slug == here.Slug {
		return nil
	}
	other, err := h.accounts.OrganizationBySlug(c.Request.Context(), slug)
	if err != nil {
		// A stale cookie naming no organization: nothing to be logged in to.
		return nil
	}
	back := h.site.OrgURL(here.Slug) + "login"
	return gin.H{
		"name":       other.Name,
		"logout_url": h.site.OrgURL(other.Slug) + "logout?return=" + url.QueryEscape(back),
	}
}

type emailBody struct {
	Email string `json:"email"`
}

// findOrganization is screen A1's lookup (4A.7). It answers the same whatever
// it finds.
func (h *handlers) findOrganization(c *gin.Context) {
	if h.resolve(c).Kind != site.General {
		abort(c, http.StatusNotFound, "unknown_address", "This lookup is at the general address.")
		return
	}
	var body emailBody
	_ = c.ShouldBindJSON(&body)
	if err := h.accounts.FindOrganizations(c.Request.Context(), body.Email); err != nil {
		log.Printf("find organization: %v", err)
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "If any organization has an account under this address, we've emailed you a link to each one."})
}

// matchOrganizations offers A1's Organization field the organizations that
// match what was typed, each with its login address.
func (h *handlers) matchOrganizations(c *gin.Context) {
	if h.resolve(c).Kind != site.General {
		abort(c, http.StatusNotFound, "unknown_address", "This list is at the general address.")
		return
	}
	orgs, err := h.accounts.MatchOrganizations(c.Request.Context(), c.Query("q"))
	if err != nil {
		h.internal(c, err)
		return
	}
	matches := make([]gin.H, len(orgs))
	for i, o := range orgs {
		matches[i] = gin.H{"name": o.Name, "slug": o.Slug, "login_url": h.site.OrgURL(o.Slug) + "login"}
	}
	c.JSON(http.StatusOK, gin.H{"organizations": matches})
}

type registerBody struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *handlers) register(c *gin.Context) {
	var body registerBody
	if err := c.ShouldBindJSON(&body); err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "The form did not arrive. Try again.")
		return
	}
	acc, err := h.accounts.Register(c.Request.Context(), organization(c), body.Email, body.Username, body.Password)
	if h.fieldErrors(c, err) {
		return
	}
	if err != nil && acc == nil {
		h.internal(c, err)
		return
	}
	if err != nil {
		// The account exists; only the email failed. A4 offers to send it
		// again.
		log.Printf("register: confirmation email: %v", err)
	}
	c.JSON(http.StatusCreated, gin.H{"email": acc.Email})
}

func (h *handlers) resendConfirmation(c *gin.Context) {
	var body emailBody
	_ = c.ShouldBindJSON(&body)
	if err := h.accounts.ResendConfirmation(c.Request.Context(), organization(c), body.Email); err != nil {
		log.Printf("resend confirmation: %v", err)
	}
	c.JSON(http.StatusAccepted, gin.H{})
}

type tokenBody struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *handlers) confirmEmail(c *gin.Context) {
	var body tokenBody
	_ = c.ShouldBindJSON(&body)
	err := h.accounts.ConfirmEmail(c.Request.Context(), organization(c), body.Token)
	if errors.Is(err, accounts.ErrLinkInvalid) {
		abort(c, http.StatusGone, "link_ended", err.Error())
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

// linkPurpose lets screen A6 show E2 before the person types a password, and
// otherwise name the account whose password is set.
func (h *handlers) linkPurpose(c *gin.Context) {
	purpose, login, err := h.accounts.LinkPurpose(c.Request.Context(), organization(c), c.Query("token"))
	if errors.Is(err, accounts.ErrLinkInvalid) {
		abort(c, http.StatusGone, "link_ended", err.Error())
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"purpose": purpose, "login": login})
}

type loginBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *handlers) forgotPassword(c *gin.Context) {
	var body loginBody
	_ = c.ShouldBindJSON(&body)
	if err := h.accounts.RequestReset(c.Request.Context(), organization(c), body.Login); err != nil {
		log.Printf("forgot password: %v", err)
	}
	c.JSON(http.StatusAccepted, gin.H{})
}

func (h *handlers) resetPassword(c *gin.Context) {
	var body tokenBody
	_ = c.ShouldBindJSON(&body)
	err := h.accounts.SetPassword(c.Request.Context(), organization(c), body.Token, body.Password)
	if errors.Is(err, accounts.ErrLinkInvalid) {
		abort(c, http.StatusGone, "link_ended", err.Error())
		return
	}
	if h.fieldErrors(c, err) {
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func (h *handlers) login(c *gin.Context) {
	org := organization(c)
	// 4A.5, rule 2: a browser logged in to another organization logs out of
	// it first.
	if other := h.otherOrganization(c, org); other != nil {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"code":                "logged_in_elsewhere",
			"message":             "You are logged in to " + other["name"].(string) + ".",
			"logged_in_elsewhere": other,
		})
		return
	}
	var body loginBody
	_ = c.ShouldBindJSON(&body)
	sess, err := h.accounts.Login(c.Request.Context(), org, body.Login, body.Password)
	switch {
	case errors.Is(err, accounts.ErrBadCredentials):
		abort(c, http.StatusUnauthorized, "bad_credentials", err.Error())
		return
	case errors.Is(err, accounts.ErrTooManyAttempts):
		abort(c, http.StatusTooManyRequests, "too_many_attempts", err.Error())
		return
	case errors.Is(err, accounts.ErrEmailNotConfirmed):
		abort(c, http.StatusForbidden, "email_not_confirmed", err.Error())
		return
	case err != nil:
		h.internal(c, err)
		return
	}
	h.setSessionCookies(c, org, sess.RefreshToken)
	c.JSON(http.StatusOK, sess)
}

func (h *handlers) refresh(c *gin.Context) {
	org := organization(c)
	raw, err := c.Cookie(sessionCookie)
	if err != nil || raw == "" {
		abort(c, http.StatusUnauthorized, "session_ended", accounts.ErrSessionEnded.Error())
		return
	}
	sess, err := h.accounts.Refresh(c.Request.Context(), org, raw)
	if errors.Is(err, accounts.ErrSessionEnded) {
		h.clearSessionCookies(c)
		abort(c, http.StatusUnauthorized, "session_ended", err.Error())
		return
	}
	if err != nil {
		h.internal(c, err)
		return
	}
	h.setSessionCookies(c, org, sess.RefreshToken)
	c.JSON(http.StatusOK, sess)
}

// logout ends the session and deletes both cookies (4A.5, rule 3). It
// succeeds with no session too, so a stale pe_org can always be cleared.
func (h *handlers) logout(c *gin.Context) {
	raw, _ := c.Cookie(sessionCookie)
	if err := h.accounts.Logout(c.Request.Context(), raw); err != nil {
		h.internal(c, err)
		return
	}
	h.clearSessionCookies(c)
	c.JSON(http.StatusOK, gin.H{})
}

func (h *handlers) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"account": c.MustGet(ctxAccount), "organization": organization(c)})
}

// resolve reads which address the request is for. On a development Mac, the
// Mac's Wi-Fi address stands for one organization, so a phone can reach it
// (config.Site.LANAddress); everywhere else the name decides.
func (h *handlers) resolve(c *gin.Context) site.Address {
	if h.atLANAddress(c) {
		return site.Address{Kind: site.Organization, Slug: h.site.LANOrganization}
	}
	return site.Resolve(c.Request.Host, h.site.OrgDomain)
}

func (h *handlers) atLANAddress(c *gin.Context) bool {
	return h.site.LANAddress != "" && h.site.LANOrganization != "" &&
		strings.EqualFold(c.Request.Host, h.site.LANAddress)
}

// orgCookieDomain is the organization cookie's domain: every organization's
// address (4A.5), or at the Wi-Fi address only that address.
func (h *handlers) orgCookieDomain(c *gin.Context) string {
	if h.atLANAddress(c) {
		return ""
	}
	return h.site.OrgDomain
}

func (h *handlers) setSessionCookies(c *gin.Context, org *accounts.Organization, refresh string) {
	maxAge := int(token.RefreshLifetime.Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, refresh, maxAge, sessionCookiePath, "", h.site.CookieSecure, true)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(orgCookie, org.Slug, maxAge, "/", h.orgCookieDomain(c), h.site.CookieSecure, true)
}

func (h *handlers) clearSessionCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, "", -1, sessionCookiePath, "", h.site.CookieSecure, true)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(orgCookie, "", -1, "/", h.orgCookieDomain(c), h.site.CookieSecure, true)
}

// fieldErrors answers 422 with the problem under each field (screen E4) and
// reports whether it did.
func (h *handlers) fieldErrors(c *gin.Context, err error) bool {
	var fields accounts.FieldErrors
	if !errors.As(err, &fields) {
		return false
	}
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
		"code":    "invalid_fields",
		"message": "Check the fields marked below.",
		"fields":  fields,
	})
	return true
}

func (h *handlers) internal(c *gin.Context, err error) {
	log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	abort(c, http.StatusInternalServerError, "internal_error", "Something went wrong on our side. Try again.")
}

func abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"code": code, "message": message})
}

func organization(c *gin.Context) *accounts.Organization {
	return c.MustGet(ctxOrganization).(*accounts.Organization)
}
