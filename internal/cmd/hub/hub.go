package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/internal/ghupdate"
	"github.com/henrygd/beszel/internal/hub"
	_ "github.com/henrygd/beszel/internal/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/spf13/cobra"
)

func main() {
	// handle health check first to prevent unneeded execution
	if len(os.Args) > 3 && os.Args[1] == "health" {
		url := os.Args[3]
		if err := checkHealth(url); err != nil {
			log.Fatal(err)
		}
		fmt.Print("ok")
		return
	}

	baseApp := getBaseApp()

	// The update command doesn't use the database, so run it without bootstrapping
	// PocketBase. Bootstrapping would create the data dir relative to the working
	// directory and open the database as the updater's user (usually root).
	if isUpdateCmd(baseApp.RootCmd, os.Args[1:]) {
		if err := baseApp.RootCmd.Execute(); err != nil {
			log.Fatal(err)
		}
		return
	}

	hub := hub.NewHub(baseApp)
	if err := hub.StartHub(); err != nil {
		log.Fatal(err)
	}
}

// getBaseApp creates a new PocketBase app with the default config
func getBaseApp() *pocketbase.PocketBase {
	isDev := os.Getenv("ENV") == "dev"

	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("POSTGRES_URL")
	}
	if databaseURL == "" {
		databaseURL = os.Getenv("BESZEL_HUB_DATABASE_URL")
	}

	// Use custom DBConnect for PostgreSQL support
	// ensurePostgresFunctions creates the pgcrypto extension and uuid_generate_v7
	// function before migrations run, since CREATE EXTENSION cannot reliably run
	// inside the migration transaction.
	if err := ensurePostgresFunctions(databaseURL); err != nil {
		log.Fatalf("failed to ensure postgres functions: %v", err)
	}

	baseApp := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: beszel.AppName + "_data",
		DefaultDev:     isDev,
		DBConnect:      dbConnect(databaseURL),
	})
	baseApp.RootCmd.Version = beszel.Version
	baseApp.RootCmd.Use = beszel.AppName
	baseApp.RootCmd.Short = ""
	// add update command
	updateCmd := &cobra.Command{
		Use:    "update",
		Short:  "Update " + beszel.AppName + " to the latest version",
		Run:    hub.Update,
		Hidden: ghupdate.SelfUpdateDisabled,
	}
	updateCmd.Flags().Bool("china-mirrors", false, "Use mirror (gh.beszel.dev) instead of GitHub")
	baseApp.RootCmd.AddCommand(updateCmd)
	// add health command
	baseApp.RootCmd.AddCommand(newHealthCmd())

	// enable auto creation of migration files when making collection changes in the Admin UI
	migratecmd.MustRegister(baseApp, baseApp.RootCmd, migratecmd.Config{
		Automigrate: isDev,
		Dir:         "../../migrations",
	})

	return baseApp
}

// isUpdateCmd reports whether args resolve to the update command.
func isUpdateCmd(rootCmd *cobra.Command, args []string) bool {
	cmd, _, err := rootCmd.Find(args)
	return err == nil && cmd.Name() == "update"
}

func newHealthCmd() *cobra.Command {
	var baseURL string

	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check health of running hub",
		Run: func(cmd *cobra.Command, args []string) {
			if err := checkHealth(baseURL); err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		},
	}
	healthCmd.Flags().StringVar(&baseURL, "url", "", "base URL")
	healthCmd.MarkFlagRequired("url")
	return healthCmd
}

// checkHealth checks the health of the hub.
func checkHealth(baseURL string) error {
	client := &http.Client{
		Timeout: time.Second * 3,
	}
	healthURL := baseURL + "/api/health"
	resp, err := client.Get(healthURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("%s returned status %d", healthURL, resp.StatusCode)
	}
	return nil
}

// dbConnect returns a DBConnectFunc that connects to PostgreSQL if databaseURL is provided,
// otherwise falls back to SQLite.
func dbConnect(databaseURL string) core.DBConnectFunc {
	return func(dbPath string) (*dbx.DB, error) {
		if databaseURL == "" {
			// default to sqlite
			return dbx.Open("sqlite", dbPath+sqlitePragmas)
		}

		// parse the URL to determine the driver and connection string
		parsedURL, err := url.Parse(databaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
		}

		scheme := strings.ToLower(parsedURL.Scheme)
		var driver, dsn string
		switch scheme {
		case "postgres", "postgresql":
			driver = "pgx"
			dsn = databaseURL // pgx accepts the full URL
		case "sqlite":
			driver = "sqlite"
			dsn = parsedURL.Path
		default:
			return nil, fmt.Errorf("unsupported database scheme %q in DATABASE_URL", scheme)
		}

		return dbx.Open(driver, dsn)
	}
}

// sqlitePragmas are the SQLite pragmas used by PocketBase's default DB connect.
const sqlitePragmas = "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=journal_size_limit(200000000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-32000)&_defensive=1"

// ensurePostgresFunctions creates the pgcrypto extension and uuid_generate_v7
// function before migrations run, since CREATE EXTENSION cannot reliably run
// inside the migration transaction.
func ensurePostgresFunctions(databaseURL string) error {
	if databaseURL == "" {
		return nil
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return err
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "postgres" && scheme != "postgresql" {
		return nil
	}

	db, err := dbx.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open postgres for extension setup: %w", err)
	}
	defer db.Close()

	if _, err := db.NewQuery(`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`).Execute(); err != nil {
		return fmt.Errorf("create pgcrypto extension: %w", err)
	}

	_, err = db.NewQuery(`
		CREATE COLLATION IF NOT EXISTS "nocase" (
			provider = icu,
			locale = 'und-u-ks-level2',
			deterministic = false
		);

		CREATE OR REPLACE FUNCTION hex(data bytea)
		RETURNS text
		LANGUAGE SQL
		IMMUTABLE
		AS $$
		SELECT encode(data, 'hex')
		$$;

		CREATE OR REPLACE FUNCTION randomblob(length integer)
		RETURNS bytea
		LANGUAGE SQL
		IMMUTABLE
		AS $$
		SELECT gen_random_bytes(length)
		$$;

		CREATE OR REPLACE FUNCTION uuid_generate_v7()
		RETURNS uuid
		LANGUAGE plpgsql
		IMMUTABLE
		AS $$
		BEGIN
			RETURN encode(
				set_bit(
					set_bit(
						overlay(uuid_send(gen_random_uuid())
								placing substring(int8send(floor(extract(epoch from clock_timestamp()) * 1000)::bigint) from 3)
								from 1 for 6
						),
						52, 1
					),
					53, 1
				),
				'hex')::uuid;
		END;
		$$;
	`).Execute()
	return err
}