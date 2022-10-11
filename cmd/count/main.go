package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.Connect(ctx, os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("failed to connect database", err)
	}
	defer db.Close()

	var now time.Time
	err = db.QueryRow(ctx, "SELECT NOW()").Scan(&now)
	if err != nil {
		log.Fatal("failed to execute query", err)
	}
	fmt.Println(now)
}
