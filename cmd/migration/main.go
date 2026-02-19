package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	steps := flag.Int("steps", 0, "number of steps to migrate (0 = all)")
	flag.Parse()

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN is required")
	}

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	default:
		log.Fatalf("unknown direction: %s", *direction)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("migration completed")
}
