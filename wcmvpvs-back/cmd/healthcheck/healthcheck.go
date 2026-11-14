/*
Healthcheck is a simple program that sends an HTTP request to the local host (self) to a configured port number.
It's used in environment where you need a simple probe for health checks (e.g., an empty container in docker).
The probe URL is http://localhost:3000/liveness . Only the port can be changed.

Usage:

	healthcheck [flags]

The flags are:

	-port <1-65535>
		Change the port where the request is sent.

Return values (exit codes):

	0
		The request was successful (HTTP 200 or HTTP 204)

	> 0
		The request was not successful (connection error or unexpected HTTP status code)
*/
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var port = flag.Int("port", 3000, "HTTP port for healthcheck")
	target := flag.String("url", "", "Health probe URL")
	dbFile := flag.String("database", "", "SQLite file for continuous Matchday probe recording")

	flag.Parse()
	if *target == "" {
		*target = fmt.Sprintf("http://localhost:%d/liveness", *port)
	}
	client := &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	if *dbFile != "" {
		monitorMatchday(client, *target, *dbFile)
		return
	}

	if err := checkHealth(client, *target); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkHealth(client *http.Client, target string) error {
	res, err := client.Get(target)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("healthcheck returned %s", res.Status)
	}
	_, err = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
	return err
}

func monitorMatchday(client *http.Client, target, dbFile string) {
	interval := time.Duration(database.MatchdayProbeIntervalSeconds) * time.Second
	// A process started mid-slot cannot observe readiness at that slot's boundary.
	time.Sleep(time.Until(time.Now().Truncate(interval).Add(interval)))
	for {
		now := time.Now()
		slot := now.Truncate(interval)
		if err := saveMatchdayProbe(dbFile, slot, checkHealth(client, target) == nil); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "cannot save Matchday health probe:", err)
		}
		time.Sleep(time.Until(slot.Add(interval)))
	}
}

func saveMatchdayProbe(dbFile string, slot time.Time, healthy bool) error {
	if _, err := os.Stat(dbFile); err != nil {
		return err
	}
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.EnsureMatchdayHealthProbes(db); err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO matchday_health_probes(slot_start,healthy) VALUES(?,?) ON CONFLICT(slot_start) DO NOTHING`, slot.Unix(), healthy)
	return err
}
