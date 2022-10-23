package migrations

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/golang-migrate/migrate/v4/database/cockroachdb"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
)

var (
	//go:embed *.sql
	files           embed.FS
	migrationSource source.Driver
)

func panicOnErr(err error) {
	if err == migrate.ErrNoChange {
		return
	}

	if err != nil {
		panic(fmt.Errorf("db/migrations: %w", err))
	}
}

func init() {
	var err error
	migrationSource, err = iofs.New(files, ".")
	panicOnErr(err)
}

/*
func dbURL() string {
	return strings.Replace(os.Getenv(DB_URL), "postgresql", "cockroachdb", 1)
}
*/

func Up(dsn string) {
	m, err := migrate.NewWithSourceInstance("embed", migrationSource, dsn)
	panicOnErr(err)
	panicOnErr(m.Up())
}

func Down(dsn string) {
	m, err := migrate.NewWithSourceInstance("embed", migrationSource, dsn)
	panicOnErr(err)

	if err = m.Down(); err == migrate.ErrNoChange {
		return
	}

	panicOnErr(err)
}
