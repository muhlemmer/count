package migrations

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func Test_panicOnErr(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "nil",
			args:    args{nil},
			wantErr: false,
		},
		{
			name:    "error",
			args:    args{errors.New("foo")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				err, _ := recover().(error)
				if tt.wantErr != (err != nil) {
					t.Errorf("panicOnErr err = %v, wantErr %v", err, tt.wantErr)
				}
			}()

			panicOnErr(tt.args.err)
		})
	}
}

// Database configuration
const (
	MigrDriverEnvKey  = "MIGRATION_DRIVER"
	DefaultMigrDriver = "pgx"
	DSNEnvKey         = "DB_URL"
	DefaultDSN        = "postgresql://muhlemmer@db:5432/muhlemmer?sslmode=disable"
)

func TestDownUp(t *testing.T) {
	migrDriver, ok := os.LookupEnv(MigrDriverEnvKey)
	if !ok {
		migrDriver = DefaultMigrDriver
	}
	dsn, ok := os.LookupEnv(DSNEnvKey)
	if !ok {
		dsn = DefaultDSN
	}

	dsn = strings.Replace(dsn, "postgresql", migrDriver, 1)

	Down(dsn)
	Up(dsn)
	Down(dsn)
}
