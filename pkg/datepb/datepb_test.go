package datepb

import (
	"fmt"
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

func TestInterval(t *testing.T) {
	type args struct {
		date *date.Date
	}
	tests := []struct {
		name      string
		args      args
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name: "year",
			args: args{&date.Date{
				Year: 1986,
			}},
			wantStart: time.Date(1986, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(1986, time.December, 31, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name: "month",
			args: args{&date.Date{
				Year:  1986,
				Month: 3,
			}},
			wantStart: time.Date(1986, time.March, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(1986, time.March, 31, 23, 59, 59, 999999999, time.UTC),
		},
		{
			name: "day",
			args: args{&date.Date{
				Year:  1986,
				Month: 3,
				Day:   25,
			}},
			wantStart: time.Date(1986, time.March, 25, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(1986, time.March, 25, 23, 59, 59, 999999999, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := Interval(tt.args.date)
			if !reflect.DeepEqual(gotStart, tt.wantStart) {
				t.Errorf("Interval() gotStart = %v, want %v", gotStart, tt.wantStart)
			}
			if !reflect.DeepEqual(gotEnd, tt.wantEnd) {
				t.Errorf("Interval() gotEnd = %v, want %v", gotEnd, tt.wantEnd)
			}
		})
	}
}

func ExampleInterval_year() {
	start, end := Interval(&date.Date{
		Year: 1986,
	})

	fmt.Println(start, end)
	// output: 1986-01-01 00:00:00 +0000 UTC 1986-12-31 23:59:59.999999999 +0000 UTC
}

func ExampleInterval_month() {
	start, end := Interval(&date.Date{
		Year:  1986,
		Month: 3,
	})

	fmt.Println(start, end)
	// output: 1986-03-01 00:00:00 +0000 UTC 1986-03-31 23:59:59.999999999 +0000 UTC
}

func ExampleInterval_day() {
	start, end := Interval(&date.Date{
		Year:  1986,
		Month: 3,
		Day:   25,
	})

	fmt.Println(start, end)
	// output: 1986-03-25 00:00:00 +0000 UTC 1986-03-25 23:59:59.999999999 +0000 UTC
}
