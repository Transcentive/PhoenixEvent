// The PhoenixEvent API server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Transcentive/PhoenixEvent/backend/internal/config"
	"github.com/Transcentive/PhoenixEvent/backend/internal/database"
	"github.com/Transcentive/PhoenixEvent/backend/internal/email"
	"github.com/Transcentive/PhoenixEvent/backend/internal/server"
	"github.com/Transcentive/PhoenixEvent/backend/internal/storage"
)

// build is the commit the binary was built from, set by the deploy script.
var build = "dev"

func main() {
	if err := run(); err != nil {
		log.Fatalf("phoenixevent: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration:\n%w", err)
	}

	// Migrations run before anything is served, and a failure stops the
	// process, so a bad migration shows as a failed service and never as a
	// server answering on a half-built schema.
	if err := database.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Both adapters are built at startup so that a host with a bad storage
	// path or email setting fails here. Nothing uses them until Phase 2.
	files, err := storage.NewLocalFS(cfg.StoragePath)
	if err != nil {
		return err
	}
	mailer := newSender(cfg.Email)
	_, _ = files, mailer

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	srv := &http.Server{
		// Localhost only: nginx is the one process open to the internet
		// (spec 2A.3).
		Addr:              fmt.Sprintf("127.0.0.1:%d", cfg.Port),
		Handler:           server.New(pool, build),
		ReadHeaderTimeout: 10 * time.Second,
	}

	failed := make(chan error, 1)
	go func() {
		log.Printf("phoenixevent %s listening on %s", build, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			failed <- err
		}
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	log.Print("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func newSender(cfg config.Email) email.Sender {
	if cfg.Host == "" {
		log.Print("email: no SMTP server configured; messages are written to this log")
		return email.LogSender{}
	}
	log.Printf("email: sending through %s:%d as %s", cfg.Host, cfg.Port, cfg.From)
	return email.SMTPSender{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		From:     cfg.From,
	}
}
