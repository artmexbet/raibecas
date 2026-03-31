package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	postgresmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed 001_create_chat_tables.up.sql 001_create_chat_tables.down.sql
var files embed.FS

func Up(databaseDSN string) error {
	db, err := sql.Open("pgx", databaseDSN)
	if err != nil {
		return fmt.Errorf("open migration database connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping migration database connection: %w", err)
	}

	sourceDriver, err := iofs.New(files, ".")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	databaseDriver, err := postgresmigrate.WithInstance(db, &postgresmigrate.Config{})
	if err != nil {
		return fmt.Errorf("create migration database driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", databaseDriver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	runErr := migrator.Up()
	sourceErr, databaseErr := migrator.Close()
	closeErr := errors.Join(sourceErr, databaseErr)

	if runErr != nil && !errors.Is(runErr, migrate.ErrNoChange) {
		return errors.Join(fmt.Errorf("apply migrations: %w", runErr), closeErr)
	}

	if closeErr != nil {
		return fmt.Errorf("close migrator: %w", closeErr)
	}

	return nil
}
