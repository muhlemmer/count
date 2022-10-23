package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/muhlemmer/count/internal/db"
	"github.com/muhlemmer/count/internal/db/migrations"
	"github.com/muhlemmer/count/internal/service"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Database configuration
const (
	MigrDriverEnvKey  = "MIGRATION_DRIVER"
	DefaultMigrDriver = "pgx"
	DSNEnvKey         = "DB_URL"
	DefaultDSN        = "postgresql://muhlemmer@db:5432/muhlemmer?sslmode=disable"
)

func run() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGHUP)
	defer cancel()

	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)

	migrDriver, ok := os.LookupEnv(MigrDriverEnvKey)
	if !ok {
		migrDriver = DefaultMigrDriver
	}
	dsn, ok := os.LookupEnv(DSNEnvKey)
	if !ok {
		dsn = DefaultDSN
	}

	migrDSN := strings.Replace(dsn, "postgresql", migrDriver, 1)
	migrations.Up(migrDSN)

	db, err := db.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	server := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	service.NewCountService(server, db)

	lis, err := net.Listen("tcp", ":7777")
	if err != nil {
		panic(err)
	}

	ec := make(chan error, 1)

	go func() {
		ec <- server.Serve(lis)
	}()
	logger.Info().Stringer("addr", lis.Addr()).Msg("grpc server listening")

	select {
	case <-ctx.Done():
		server.GracefulStop()
	case err = <-ec:
		logger.Panic().Err(err).Msg("grpc server terminated unexpectedly")
	}

	err = <-ec
	logger.Err(err).Msg("grpc server terminated")
	if err != nil {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
