package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ItsKevinRafaell/go-momentum-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnection(ctx context.Context) *pgxpool.Pool {
	// Ambil URL database dari .env
	dbUrl := config.Get("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Buat koneksi pool baru
	dbPool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}

	// Lakukan ping untuk memastikan koneksi berhasil
	if err := dbPool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	return dbPool
}