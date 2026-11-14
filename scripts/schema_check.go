package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pocketbase/dbx"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:Password09@localhost:5006/server?sslmode=disable"
	}
	db, err := dbx.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tables := []string{
		"users", "systems", "containers", "container_stats", "system_stats",
		"system_details", "systemd_services", "smart_devices", "zfs_pools",
		"network_monitors", "network_monitor_stats", "alerts", "alerts_history",
		"user_settings", "quiet_hours", "fingerprints", "universal_tokens",
	}
	for _, t := range tables {
		var cols []struct {
			Column string `db:"column_name"`
			Type   string `db:"data_type"`
			Null   string `db:"is_nullable"`
		}
		err := db.NewQuery("select column_name, data_type, is_nullable from information_schema.columns where table_schema='public' and table_name={:t} order by ordinal_position").
			Bind(dbx.Params{"t": t}).All(&cols)
		if err != nil {
			fmt.Printf("TABLE %s: ERR %v\n", t, err)
			continue
		}
		fmt.Printf("TABLE %s (%d cols)\n", t, len(cols))
		for _, c := range cols {
			fmt.Printf("  - %s %s null=%s\n", c.Column, c.Type, c.Null)
		}
	}
}
