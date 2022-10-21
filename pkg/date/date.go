package date

import (
	"time"

	"google.golang.org/genproto/googleapis/type/date"
)

// Time converts a googleapis/type/date.Date to a standard time.Time in UTC timezone.
// A date can represent:
//   - a full year when month and day are empty.
//     The time will be 1st of january 00:00 UTC in the given year.
//   - a full month when day is empty.
//   - an exact date with year, month and day.
//
// Overlowing or negative values are normalized by Go's time.Date() function.
func Time(date *date.Date) time.Time {
	var (
		year  = date.GetYear()
		month = date.GetMonth()
		day   = date.GetDay()
	)

	if day == 0 {
		day = 1
		if month == 0 {
			month = 1
		}
	}

	return time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
}

// Date converts a standard time.Time to a googleapis/type/date.Date in UTC timezone.
func Date(ts time.Time) *date.Date {
	year, month, day := ts.UTC().Date()
	return &date.Date{
		Year:  int32(year),
		Month: int32(month),
		Day:   int32(day),
	}
}

// Today returns today's date.
func Today() *date.Date {
	return Date(time.Now())
}
