package config

import (
	"database/sql"
	"os"

	"github.com/joho/godotenv"
	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
)

func LoadAppConfig() (AppConfig, error) {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		return AppConfig{}, err
	}

	return AppConfig{
		JwtSecret: os.Getenv("JWT_SECRET"),
		PolkaKey:  os.Getenv("POLKA_KEY"),
		DbQueries: *database.New(db),
	}, nil
}
