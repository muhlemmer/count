package db

import (
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	countv1 "github.com/muhlemmer/count/pkg/api/count/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// scanMethodCountRows scans Rows into a slice of *countv1.MethodCount.
func scanMethodCountRows(rows pgx.Rows) (results []*countv1.MethodCount, err error) {
	for rows.Next() {
		var (
			day    pgtype.Date
			method pgtype.Varchar
			path   pgtype.Varchar
			total  pgtype.Int4
		)

		if err = rows.Scan(&day, &method, &path, &total); err != nil {
			return nil, err
		}

		results = append(results, &countv1.MethodCount{
			Date:   timestamppb.New(day.Time),
			Method: countv1.Method(countv1.Method_value[method.String]),
			Path:   path.String,
			Count:  total.Int,
		})
	}

	return results, rows.Err()
}
