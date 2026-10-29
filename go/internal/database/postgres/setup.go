package postgres

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	once       sync.Once
	dbInstance *gorm.DB
	initErr    error
)

// GetTestDB spins up the container on the very first test call
// and reuses it for all subsequent tests (whether run via CLI or VS Code UI).
func GetTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	once.Do(func() {
		ctx := context.Background()

		pgContainer, err := postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("test_db"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			initErr = err
			return
		}

		// Note: Testcontainers handles container termination automatically
		// via its Ryuk sidecar when your test process exits.

		connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			initErr = err
			return
		}

		dbInstance, err = gorm.Open(pg.Open(connStr), &gorm.Config{})
		if err != nil {
			initErr = err
			return
		}

		// Run AutoMigrate globally just once when the container boots
		initErr = dbInstance.AutoMigrate(
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
	})

	require.NoError(t, initErr, "failed to initialize test database container")
	return dbInstance
}
