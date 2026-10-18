package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	// 1. Spin up the official Postgres container instance
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres container")

	// 2. Register automatic teardown via t.Cleanup()
	// This guarantees the container is destroyed when the test finishes,
	// even if the test fails or panics midway.
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(pgContainer)
		require.NoError(t, err, "failed to terminate postgres container")
	})

	// 3. Retrieve the dynamically mapped connection string (ephemeral host port)
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	// 4. Connect via GORM
	db, err := gorm.Open(pg.Open(connStr), &gorm.Config{})
	require.NoError(t, err, "failed to connect to database via gorm")

	// 5. Automatically migrate your GORM schemas on the fly
	err = db.AutoMigrate(
		&Vulnerability{},
		&AffectedVersions{},
		&AffectedCommits{},
		&AliasGroup{},
		&UpstreamGroup{},
		&RelatedGroup{},
		&AliasAllowListEntry{},
		&AliasDenyListEntry{},
		&RepoAllowList{},
		&ListedVulnerability{},
		&SourceRepository{},
		&RepoIndex{},
		&RepoIndexBucket{},
	)
	require.NoError(t, err, "failed to run database auto-migration")

	return db
}
