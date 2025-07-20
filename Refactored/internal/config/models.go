package config

import (
	"sync/atomic"

	"github.com/mikedarling/BootDotDev-GoHttpServer/internal/database"
)

type AppConfig struct {
	FileServerHits atomic.Int32
	DbQueries      database.Queries
	JwtSecret      string
	PolkaKey       string
}
