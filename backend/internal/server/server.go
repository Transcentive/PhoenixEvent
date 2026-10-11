// Package server builds the HTTP API. Every route is under /api/v1, which is
// the one path nginx passes through to this process.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger reports whether the database answers. The connection pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// New returns the router. build identifies the commit the binary was built
// from, so a health check also says which build is running.
func New(db Pinger, build string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	// nginx is the only caller (spec 2A.3), so it is the only proxy whose
	// forwarded address is believed.
	_ = router.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	v1 := router.Group("/api/v1")
	v1.GET("/health", health(db, build))

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
