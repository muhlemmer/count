package db

import _ "embed"

var (
	//go:embed queries/insert_method_request.sql
	insertMethodRequestSQL string
	//go:embed queries/count_daily_method_totals.sql
	countDailyMethodTotalsSQL string
	//go:embed queries/list_daily_totals_interval.sql
	listDailyTotalsSQL string
	//go:embed queries/get_period_totals.sql
	getPeriodTotalsSQL string
)
