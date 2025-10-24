/*
Webapi is the executable for the main web server.
It builds a web server around APIs from `service/api`.
Webapi connects to external resources needed (database) and starts two web servers: the API web server, and the debug.
Everything is served via the API web server, except debug variables (/debug/vars) and profiler infos (pprof).

Usage:

	webapi [flags]

Flags and configurations are handled automatically by the code in `load-configuration.go`.

Return values (exit codes):

	0
		The program ended successfully (no errors, stopped by signal)

	> 0
		The program ended due to an error

Note that this program will update the schema of the database to the latest version available (embedded in the
executable during the build).
*/
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/globaltime"
	"github.com/ardanlabs/conf"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

func ensureBootstrapAdmin(db database.AppDatabase, logger *logrus.Logger, cfg WebAPIConfiguration) error {
	if !cfg.BootstrapAdmin.Enabled {
		logger.Info("bootstrap admin disabilitato via configurazione")
		return nil
	}

	username := strings.TrimSpace(cfg.BootstrapAdmin.Username)
	passhash := strings.TrimSpace(cfg.BootstrapAdmin.PasswordHash)
	role := strings.TrimSpace(cfg.BootstrapAdmin.Role)
	if username == "" || passhash == "" {
		logger.Warn("bootstrap admin non configurato: username o password hash mancanti")
		return nil
	}

	if role == "" {
		role = "staff"
	}

	// Se esiste già, non fare nulla
	if _, err := db.GetAdminByUsername(username); err == nil {
		logger.Infof("admin %q già presente, skip bootstrap", username)
		return nil
	}

	// Prova a crearlo
	id, err := db.CreateAdmin(database.Admin{
		Username:     username,
		PasswordHash: passhash,
		Role:         role,
	})
	if err != nil {
		return fmt.Errorf("creazione admin bootstrap: %w", err)
	}
	logger.Infof("admin bootstrap creato: %s (id=%d, role=%s)", username, id, role)
	return nil
}

// main is the program entry point. The only purpose of this function is to call run() and set the exit code if there is
// any error
func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error: ", err)
		os.Exit(1)
	}
}

// run executes the program. The body of this function should perform the following steps:
// * reads the configuration
// * creates and configure the logger
// * connects to any external resources (like databases, authenticators, etc.)
// * creates an instance of the service/api package
// * starts the principal web server (using the service/api.Router.Handler() for HTTP handlers)
// * waits for any termination event: SIGTERM signal (UNIX), non-recoverable server error, etc.
// * closes the principal web server
func run() error {
	rand.Seed(globaltime.Now().UnixNano())
	// Load Configuration and defaults
	cfg, err := loadConfiguration()
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			return nil
		}
		return err
	}

	// Init logging
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	if cfg.Debug {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.Infof("application initializing")

	// Start Database
	logger.Println("initializing database support")
	dbconn, err := sql.Open("sqlite3", cfg.DB.Filename)
	if err != nil {
		logger.WithError(err).Error("error opening SQLite DB")
		return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
		logger.Debug("database stopping")
		_ = dbconn.Close()
	}()
	db, err := database.New(dbconn)
	if err != nil {
		logger.WithError(err).Error("error creating AppDatabase")
		return fmt.Errorf("creating AppDatabase: %w", err)
	}
	if err := ensureBootstrapAdmin(db, logger, cfg); err != nil {
		logger.WithError(err).Error("bootstrap admin fallito")
		return err
	}
	// Start (main) API server
	logger.Info("initializing API server")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	ipKey := strings.TrimSpace(os.Getenv("HMAC_IP_KEY"))
	if ipKey == "" {
		logger.Error("HMAC_IP_KEY non configurata")
		return fmt.Errorf("missing HMAC_IP_KEY environment variable")
	}

	codeKey := strings.TrimSpace(os.Getenv("HMAC_CODE_KEY"))
	if codeKey == "" {
		logger.Error("HMAC_CODE_KEY non configurata")
		return fmt.Errorf("missing HMAC_CODE_KEY environment variable")
	}

	// Create the API router
	apirouter, err := api.New(api.Config{
		Logger:      logger,
		Database:    db,
		VoteSecret:  cfg.Vote.Secret,
		HMACIPKey:   ipKey,
		HMACCodeKey: codeKey,
	})
	if err != nil {
		logger.WithError(err).Error("error creating the API server instance")
		return fmt.Errorf("creating the API server instance: %w", err)
	}
	chiRouter := apirouter.Handler()

	var handler http.Handler = chiRouter

	handler, err = registerWebUI(handler)
	if err != nil {
		logger.WithError(err).Error("error registering web UI handler")
		return fmt.Errorf("registering web UI handler: %w", err)
	}

	// Apply CORS policy
	handler = applyCORSHandler(handler)

	// Create the API server
	apiserver := http.Server{
		Addr:              cfg.Web.APIHost,
		Handler:           handler,
		ReadTimeout:       cfg.Web.ReadTimeout,
		ReadHeaderTimeout: cfg.Web.ReadTimeout,
		WriteTimeout:      cfg.Web.WriteTimeout,
	}

	// Start the service listening for requests in a separate goroutine
	go func() {
		logger.Infof("API listening on %s", apiserver.Addr)
		serverErrors <- apiserver.ListenAndServe()
		logger.Infof("stopping API server")
	}()

	// Waiting for shutdown signal or POSIX signals
	select {
	case err := <-serverErrors:
		// Non-recoverable server error
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Infof("signal %v received, start shutdown", sig)

		// Asking API server to shut down and load shed.
		err := apirouter.Close()
		if err != nil {
			logger.WithError(err).Warning("graceful shutdown of apirouter error")
		}

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		// Asking listener to shut down and load shed.
		err = apiserver.Shutdown(ctx)
		if err != nil {
			logger.WithError(err).Warning("error during graceful shutdown of HTTP server")
			if closeErr := apiserver.Close(); closeErr != nil {
				logger.WithError(closeErr).Warning("error closing HTTP server")
			}
		}

	}

	return nil
}
