package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/9904099/opsledger/internal/app"
	"github.com/9904099/opsledger/internal/store"
)

func main() {
	addr := envOrDefault("OPSLEDGER_ADDR", "127.0.0.1:18090")
	dbDriver := envOrDefault("OPSLEDGER_DB_DRIVER", "sqlite")
	dataPath := envOrDefault("OPSLEDGER_DATA", filepath.Join("data", "opsledger.db"))
	dbDSN := envOrDefault("OPSLEDGER_DB_DSN", "")

	flag.StringVar(&addr, "addr", addr, "HTTP listen address")
	flag.StringVar(&dbDriver, "db-driver", dbDriver, "database driver: sqlite, postgres, or mysql")
	flag.StringVar(&dataPath, "data", dataPath, "database file path")
	flag.StringVar(&dbDSN, "db-dsn", dbDSN, "database DSN for PostgreSQL or MySQL")
	flag.Parse()

	server, err := app.NewServerWithConfig(app.ServerConfig{
		Database: store.DatabaseConfig{
			Driver: dbDriver,
			DSN:    dbDSN,
			Path:   dataPath,
		},
	})
	if err != nil {
		log.Fatalf("initialize server: %v", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			log.Printf("close store: %v", err)
		}
	}()

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("opsledger listening on http://%s", addr)
	log.Printf("opsledger database driver: %s", dbDriver)
	if dbDriver == "sqlite" || dbDriver == "sqlite3" {
		log.Printf("opsledger database file: %s", dataPath)
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
