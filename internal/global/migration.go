package global

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

// Migrate creates a new migration instance using the provided source and database URLs.
func Migrate(srcURL, dbURL string) (*migrate.Migrate, error) {
	m, err := migrate.New(srcURL, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}
	return m, nil
}
