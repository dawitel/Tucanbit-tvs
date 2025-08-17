package main

import (
	"github.com/tuncanbit/tvs/internal/application/services"
	"github.com/tuncanbit/tvs/internal/infrastructure/database"
	"github.com/tuncanbit/tvs/internal/infrastructure/http/clients"
	transactionrepository "github.com/tuncanbit/tvs/internal/repositories/transaction_repo"
	"github.com/tuncanbit/tvs/internal/server"
	"github.com/tuncanbit/tvs/pkg/config"
	"github.com/tuncanbit/tvs/pkg/logger"
)

func main() {
	logger := logger.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.ShutDown()

	transactionRepo := transactionrepository.New(db, logger)
	ieClient := clients.NewIndexingEngineClient(cfg.ExternalServices.IndexingEngine, logger)
	pdmClient := clients.NewPDMClient(cfg.ExternalServices.PDM, logger)

	verificationService := services.NewVerificationService(
		transactionRepo,
		ieClient,
		pdmClient,
		cfg.Verification,
		logger,
	)

	srv := server.New(cfg, verificationService, logger)
	srv.Start()
}
