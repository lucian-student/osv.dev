package postgres

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	pgContainer *postgres.PostgresContainer
	once        sync.Once
	dbInstance  *gorm.DB
	initErr     error
)

// GetTestDB spins up the container on the very first test call
// and reuses it for all subsequent tests (whether run via CLI or VS Code UI).
func GetTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	once.Do(func() {
		ctx := t.Context()

		pgContainer, initErr = postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("test_db"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			postgres.BasicWaitStrategies(),
		)
		if initErr != nil {
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

func TestMain(m *testing.M) {
	code := m.Run()
	if pgContainer != nil {
		err := pgContainer.Terminate(context.Background())
		if err != nil {
			fmt.Printf("Failed to terminate testing postgres container!")
			os.Exit(1)
		}
	}
	os.Exit(code)
}
