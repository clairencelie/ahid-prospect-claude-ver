package config

import (
	"context"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds runtime settings. Thresholds are seeded into the `settings`
// table (FR3.6) and loaded here at boot; env vars are the fallback when a
// key is missing from the table (e.g. a fresh DB before seed.sql runs).
type Config struct {
	DatabaseURL     string
	Port            string
	BlockThreshold  float64
	ReviewThreshold float64
	LockTTLDays     int
}

func FromEnv() Config {
	return Config{
		DatabaseURL:     getenv("DATABASE_URL", "postgres://ahid:ahid@localhost:5432/ahid_prospect?sslmode=disable"),
		Port:            getenv("PORT", "8080"),
		BlockThreshold:  getenvFloat("BLOCK_THRESHOLD", 80),
		ReviewThreshold: getenvFloat("REVIEW_THRESHOLD", 50),
		LockTTLDays:     getenvInt("LOCK_TTL_DAYS", 30),
	}
}

// LoadSettings overlays thresholds from the `settings` table on top of the
// env-derived defaults, since the table is the authoritative source once seeded.
func (c *Config) LoadSettings(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		switch key {
		case "block_threshold":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				c.BlockThreshold = f
			}
		case "review_threshold":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				c.ReviewThreshold = f
			}
		case "lock_ttl_days":
			if i, err := strconv.Atoi(value); err == nil {
				c.LockTTLDays = i
			}
		}
	}
	return rows.Err()
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
