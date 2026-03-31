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

//go:embed 000001_create_reference_tables.up.sql 000001_create_reference_tables.down.sql 000002_create_documents_table.up.sql 000002_create_documents_table.down.sql 000003_create_document_versions_table.up.sql 000003_create_document_versions_table.down.sql 000004_seed_data.up.sql 000004_seed_data.down.sql 000005_add_cover_path.up.sql 000005_add_cover_path.down.sql 000006_create_document_bookmarks_table.up.sql 000006_create_document_bookmarks_table.down.sql
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
