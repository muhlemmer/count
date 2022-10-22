// Package datepb provides utilities for googleapis/types/date.Date conversion.
package datepb

import (
	"time"

	"google.golang.org/genproto/googleapis/type/date"
)

func extractYMD(date *date.Date) (year int, month time.Month, day int) {
	y := date.GetYear()
	m := date.GetMonth()
	d := date.GetDay()

	return int(y), time.Month(m), int(d)
}

// Time converts a googleapis/type/date.Date to a standard time.Time in UTC timezone.
// A date can represent:
//   - a full year when month and day are empty.
//     The time will be 1st of january 00:00 UTC in the given year.
//   - a full month when day is empty.
//   - an exact date with year, month and day.
//
// Overlowing or negative values are normalized by Go's time.Date() function.
func Time(date *date.Date) time.Time {
	year, month, day := extractYMD(date)

	if day == 0 {
		day = 1
		if month == 0 {
			month = 1
		}
	}

	return time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
}

// Interval returns a start and end time.Time for the given period in UTC.
// The precision of the period is determined by the populated fields.
// start is always at 00:00 on the beginning of the period.
// end is the period increased by 1, minus 1ns.
// For example:
//   - if day is non-zero the period starts at 00:00 and ends 1ns before end of day.
//   - if day is zero and month is non-zero the period starts on the first day 00:00
//     and ends 1ns before the end of the last day of the month.
//   - if day and month are zero, the period starts on 1st of January 00:00 and ends on 31st of December 23:59:00
func Interval(date *date.Date) (start, end time.Time) {
	year, month, day := extractYMD(date)

	if day == 0 {
		if month == 0 {
			return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(year+1, 1, 1, 0, 0, 0, -1, time.UTC)
		}

		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, month+1, 1, 0, 0, 0, -1, time.UTC)
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
		time.Date(year, month, day+1, 0, 0, 0, -1, time.UTC)
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
