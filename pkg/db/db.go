package db

import (
	"fmt"

	"github.com/tuncanbit/tvs/pkg/config"
)

func GetDBDSN(config *config.DatabaseConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
		config.SSLMode,
	)
}
