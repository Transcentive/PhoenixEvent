// The PhoenixEvent API server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Transcentive/PhoenixEvent/backend/internal/accounts"
	"github.com/Transcentive/PhoenixEvent/backend/internal/config"
	"github.com/Transcentive/PhoenixEvent/backend/internal/database"
	"github.com/Transcentive/PhoenixEvent/backend/internal/email"
	"github.com/Transcentive/PhoenixEvent/backend/internal/server"
	"github.com/Transcentive/PhoenixEvent/backend/internal/storage"
	"github.com/Transcentive/PhoenixEvent/backend/internal/token"
)

// build is the commit the binary was built from, set by the deploy script.
var build = "dev"

func main() {
	run := serve
	if len(os.Args) > 1 && os.Args[1] == "create-organization" {
		run = createOrganization
	}
	if err := run(); err != nil {
		log.Fatalf("phoenixevent: %v", err)
	}
}

// createOrganization is the System Administrator's command (spec 4A.6):
//
//	phoenixevent-server create-organization -name "Lincoln Elementary PTA" \
//	    -address lincolnpta -admin-email admin@example.org
//
// It runs with the same environment as the server, so it reaches the same
// database and sends email the same way.
func createOrganization() error {
	flags := flag.NewFlagSet("create-organization", flag.ContinueOnError)
	name := flags.String("name", "", "the organization's name, such as \"Lincoln Elementary PTA\"")
	address := flags.String("address", "", "the address name, such as lincolnpta; fixed once set")
	adminEmail := flags.String("admin-email", "", "the email address of the first Organization Administrator")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if *name == "" || *address == "" || *adminEmail == "" {
		flags.Usage()
		return errors.New("-name, -address and -admin-email are all required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration:\n%w", err)
	}
	if err := database.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	svc := accounts.NewService(pool, newSender(cfg.Email), cfg.Site, token.NewIssuer(cfg.JWTSecret))
	org, err := svc.CreateOrganization(ctx, *name, *address, *adminEmail)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s at %s\nThe link to set a password went to %s.\n", org.Name, cfg.Site.OrgURL(org.Slug), *adminEmail)
	return nil
}

func serve() error {
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

	// The storage adapter is built at startup so that a host with a bad
	// path fails here. Nothing uses it until uploads arrive.
	files, err := storage.NewLocalFS(cfg.StoragePath)
	if err != nil {
		return err
	}
	_ = files
	tokens := token.NewIssuer(cfg.JWTSecret)
	accountService := accounts.NewService(pool, newSender(cfg.Email), cfg.Site, tokens)

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	srv := &http.Server{
		// Localhost only: nginx is the one process open to the internet
		// (spec 2A.3).
		Addr: fmt.Sprintf("127.0.0.1:%d", cfg.Port),
		Handler: server.New(server.Deps{
			DB:       pool,
			Build:    build,
			Accounts: accountService,
			Tokens:   tokens,
			Site:     cfg.Site,
		}),
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
