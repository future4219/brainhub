package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"brainhub/config"
	brainhubmigrations "brainhub/db/migrations"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: migrate up | down <steps> | version")
	}
	databaseURL, err := config.DatabaseURL()
	if err != nil {
		log.Fatal(err)
	}
	source, err := iofs.New(brainhubmigrations.Files, ".")
	if err != nil {
		log.Fatal(err)
	}
	migrationURL, err := url.Parse(databaseURL)
	if err != nil {
		log.Fatal("parse migration database URL")
	}
	migrationURL.Scheme = "pgx5"
	runner, err := migrate.NewWithSourceInstance("iofs", source, migrationURL.String())
	if err != nil {
		log.Fatal("open migration database")
	}
	defer func() {
		_, _ = runner.Close()
	}()

	switch os.Args[1] {
	case "up":
		err = runner.Up()
	case "down":
		if len(os.Args) != 3 {
			log.Fatal("usage: migrate down <steps>")
		}
		steps, parseErr := strconv.Atoi(os.Args[2])
		if parseErr != nil || steps < 1 {
			log.Fatal("migration steps must be a positive integer")
		}
		err = runner.Steps(-steps)
	case "version":
		version, dirty, versionErr := runner.Version()
		if versionErr != nil {
			err = versionErr
			break
		}
		fmt.Printf("version=%d dirty=%t\n", version, dirty)
	default:
		log.Fatal("usage: migrate up | down <steps> | version")
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(err)
	}
}
