package migrations

import (
	"errors"
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

func TestDownUp(t *testing.T) {
	const dsn = "pgx://muhlemmer@db:5432/muhlemmer"

	Down(dsn)
	Up(dsn)
	Down(dsn)
}
