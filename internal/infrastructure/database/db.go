package database

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/tuncanbit/tvs/pkg/config"
	"github.com/tuncanbit/tvs/pkg/db"
)

type DBManager struct {
	Db *sql.DB
}

func New(cfg *config.DatabaseConfig) (*DBManager, error) {
	DBDSN := db.GetDBDSN(cfg)
	Db, err := sql.Open("postgres", DBDSN)
	if err != nil {
		return nil, err
	}
	if err := Db.Ping(); err != nil {
		return nil, err
	}

	return &DBManager{
		Db: Db,
	}, nil
}

func (dm *DBManager) ShutDown() {
	if dm.Db != nil {
		dm.Db.Close()
	}
}
