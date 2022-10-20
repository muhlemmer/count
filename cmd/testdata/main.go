package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/muhlemmer/count/internal/db"
	"github.com/muhlemmer/count/internal/db/migrations"
	"github.com/rs/zerolog"
)

const dsn = "postgresql://muhlemmer@db:5432/muhlemmer?sslmode=disable"

func run() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGHUP)
	defer cancel()

	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)

	migrDSN := strings.Replace(dsn, "postgresql", "cockroachdb", 1)

	migrations.Down(migrDSN)
	migrations.Up(migrDSN)

	db, err := db.New(ctx, dsn)
	if err != nil {
		return 1
	}
	defer db.Close()

	if err := db.InsertMethodRequestTestdata(ctx, 1000, time.Now(), time.Now().Add(24*time.Hour)); err != nil {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
