package db

import (
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/muhlemmer/count/pkg/datepb"
)

// scanMethodCountRows scans Rows into a slice of *countv1.MethodCount.
func scanMethodCountRows(rows pgx.Rows) (results []*countv1.MethodCount, err error) {
	for rows.Next() {
		var (
			date   pgtype.Date
			method pgtype.Varchar
			path   pgtype.Varchar
			total  pgtype.Int8
		)

		if err = rows.Scan(&date, &method, &path, &total); err != nil {
			return nil, err
		}

		mc := &countv1.MethodCount{
			Method: countv1.Method(countv1.Method_value[method.String]),
			Path:   path.String,
			Count:  total.Int,
		}

		if date.Status == pgtype.Present {
			mc.Date = datepb.Date(date.Time)
		}

		results = append(results, mc)
	}

	return results, rows.Err()
}
