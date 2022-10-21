package date

import (
	"reflect"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/protobuf/proto"
)

func TestTime(t *testing.T) {
	type args struct {
		date *date.Date
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{
			name: "empty",
			want: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "year",
			args: args{&date.Date{
				Year: 1986,
			}},
			want: time.Date(1986, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "month",
			args: args{&date.Date{
				Year:  1986,
				Month: 3,
			}},
			want: time.Date(1986, time.March, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "day",
			args: args{&date.Date{
				Year:  1986,
				Month: 3,
				Day:   25,
			}},
			want: time.Date(1986, time.March, 25, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Time(tt.args.date)
			if !got.Equal(tt.want) {
				t.Errorf("Time() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDate(t *testing.T) {
	want := &date.Date{
		Year:  1986,
		Month: 3,
		Day:   25,
	}
	got := Date(time.Date(1986, time.March, 25, 0, 0, 0, 0, time.UTC))

	if !proto.Equal(got, want) {
		t.Errorf("Date() = %v, want %v", got, want)
	}
}

func TestToday(t *testing.T) {
	want := Date(time.Now())
	got := Today()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Today() = %v, want %v", got, want)
	}
}
