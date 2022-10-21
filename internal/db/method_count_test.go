package db

import (
	"testing"
)

// just test the error, for coverage.
func Test_scanMethodCountRows(t *testing.T) {
	rows, err := testDB.pool.Query(testCTX, "select 1;")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	_, err = scanMethodCountRows(rows)
	if err == nil {
		t.Error("scanMethodCountRows(): expected error")
	}
}
