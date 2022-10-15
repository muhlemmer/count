package timer

import (
	"testing"
	"time"
)

func Test_randomDuration(t *testing.T) {
	type args struct {
		min time.Duration
		max time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    time.Duration // result is detirministic because math/rand
		wantErr error
	}{
		{
			name:    "negative err",
			args:    args{-1, 2},
			wantErr: errNegativeValue,
		},
		{
			name:    "min-max err",
			args:    args{3, 2},
			wantErr: errMinMax,
		},
		{
			name: "zero min",
			args: args{0, time.Second},
			want: 947779410,
		},
		{
			name: "interval",
			args: args{time.Second / 5, time.Second * 5},
			want: 2482153551,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				err, _ := recover().(error)
				if err != tt.wantErr {
					t.Errorf("randomDuration()  err = %v, want %v", err, tt.wantErr)
				}
			}()

			if got := randomDuration(tt.args.min, tt.args.max); got != tt.want {
				t.Errorf("randomDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_randomDuration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		randomDuration(time.Second/5, time.Second*5)
	}
}

func Test_RandomTimer(t *testing.T) {
	<-RandomTimer(0, time.Millisecond)
}
