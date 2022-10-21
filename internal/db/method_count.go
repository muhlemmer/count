package db

import (
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"github.com/muhlemmer/count/pkg/date"
)

// scanMethodCountRows scans Rows into a slice of *countv1.MethodCount.
func scanMethodCountRows(rows pgx.Rows) (results []*countv1.MethodCount, err error) {
	for rows.Next() {
		var (
			pdate  pgtype.Date
			method pgtype.Varchar
			path   pgtype.Varchar
			total  pgtype.Int4
		)

		if err = rows.Scan(&pdate, &method, &path, &total); err != nil {
			return nil, err
		}

		results = append(results, &countv1.MethodCount{
			Date:   date.Date(pdate.Time),
			Method: countv1.Method(countv1.Method_value[method.String]),
			Path:   path.String,
			Count:  total.Int,
		})
	}

	return results, rows.Err()
}
