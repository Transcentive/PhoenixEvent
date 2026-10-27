// Package server builds the HTTP API. Every route is under /api/v1, which is
// the one path nginx passes through to this process.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Transcentive/PhoenixEvent/backend/internal/accounts"
	"github.com/Transcentive/PhoenixEvent/backend/internal/config"
	"github.com/Transcentive/PhoenixEvent/backend/internal/token"
)

// Pinger reports whether the database answers. The connection pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps is what the routes need. Build identifies the commit the binary was
// built from, so a health check also says which build is running.
type Deps struct {
	DB       Pinger
	Build    string
	Accounts *accounts.Service
	Tokens   *token.Issuer
	Site     config.Site
}

type handlers struct {
	accounts *accounts.Service
	tokens   *token.Issuer
	site     config.Site
}

// New returns the router.
func New(d Deps) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	// nginx is the only caller (spec 2A.3), so it is the only proxy whose
	// forwarded address is believed.
	_ = router.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	v1 := router.Group("/api/v1")
	v1.GET("/health", health(d.DB, d.Build))

	if d.Accounts != nil {
		h := &handlers{accounts: d.Accounts, tokens: d.Tokens, site: d.Site}
		v1.GET("/address", h.address)
		v1.POST("/find-organization", h.findOrganization)
		v1.GET("/organizations", h.matchOrganizations)

		org := v1.Group("", h.withOrganization)
		auth := org.Group("/auth")
		auth.POST("/register", h.register)
		auth.POST("/resend-confirmation", h.resendConfirmation)
		auth.POST("/confirm-email", h.confirmEmail)
		auth.GET("/link", h.linkPurpose)
		auth.POST("/forgot-password", h.forgotPassword)
		auth.POST("/reset-password", h.resetPassword)
		auth.POST("/login", h.login)
		auth.POST("/refresh", h.refresh)
		auth.POST("/logout", h.logout)

		signedIn := org.Group("", h.requireSession)
		signedIn.GET("/me", h.me)
		signedIn.GET("/profile", h.profile)
		signedIn.PUT("/profile", h.updateProfile)
		signedIn.PUT("/profile/visibility", h.updateVisibility)
		signedIn.POST("/profile/email", h.changeEmail)
		signedIn.GET("/people/:id", h.person)
	}

	return router
}

// health answers 200 only when the process is up and the database answers, so
// one check tells a deploy that the whole path works.
func health(db Pinger, build string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unavailable",
				"database": "unreachable",
				"build":    build,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "ok",
			"build":    build,
		})
	}
}
