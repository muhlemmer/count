package tester

import _ "embed"

var (
	//go:embed queries/insert_methods.sql
	insertMethodsSQL string
	//go:embed queries/insert_requests.sql
	insertRequestsSQL string
	//go:embed queries/insert_daily_method_totals.sql
	insertDailyMethodTotalsSQL string
)
