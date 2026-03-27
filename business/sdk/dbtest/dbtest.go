package dbtest

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/casebrophy/planner/business/sdk/migrate"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/sqldb"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// New creates a new test database inside a Docker Postgres container.
// The database is migrated and a connection pool is returned with all
// business domain packages wired.
func New(t *testing.T, testName string) *Database {
	t.Helper()

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       "postgres",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("getting container host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("getting container port: %v", err)
	}

	port, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		t.Fatalf("parsing container port: %v", err)
	}

	// Connect to the default postgres database to create a test database.
	dbM, err := sqldb.Open(sqldb.Config{
		Host:       host,
		Port:       port,
		User:       "postgres",
		Password:   "postgres",
		Name:       "postgres",
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("opening admin database connection: %v", err)
	}

	connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := sqldb.StatusCheck(connCtx, dbM); err != nil {
		t.Fatalf("database status check: %v", err)
	}

	// Create a unique test database.
	const letterBytes = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 4)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	dbName := string(b)

	t.Logf("Create Database: %s\n", dbName)
	if _, err := dbM.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		t.Fatalf("creating database %s: %v", dbName, err)
	}

	// Open connection to the test database.
	db, err := sqldb.Open(sqldb.Config{
		Host:       host,
		Port:       port,
		User:       "postgres",
		Password:   "postgres",
		Name:       dbName,
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("opening test database connection: %v", err)
	}

	// Run migrations.
	migCtx, migCancel := context.WithTimeout(ctx, 30*time.Second)
	defer migCancel()

	t.Logf("Migrate Database: %s\n", dbName)
	if err := migrate.Migrate(migCtx, db); err != nil {
		t.Fatalf("migrating database: %v", err)
	}

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LevelInfo, "TEST")

	t.Cleanup(func() {
		t.Helper()

		t.Logf("Drop Database: %s\n", dbName)
		if _, err := dbM.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE %s", dbName)); err != nil {
			t.Logf("dropping database %s: %v", dbName, err)
		}

		db.Close()
		dbM.Close()

		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminating container: %v", err)
		}

		t.Logf("******************** LOGS (%s) ********************\n\n", testName)
		t.Log(buf.String())
		t.Logf("******************** LOGS (%s) ********************\n", testName)
	})

	return &Database{
		DB:        db,
		Log:       log,
		BusDomain: newBusDomains(log, db),
	}
}
