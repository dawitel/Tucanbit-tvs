package main

import (
	"context"

	authservice "github.com/tuncanbit/tvs/internal/application/auth"
	"github.com/tuncanbit/tvs/internal/application/verificationservice"
	"github.com/tuncanbit/tvs/internal/infrastructure/clients"
	"github.com/tuncanbit/tvs/internal/infrastructure/database"
	"github.com/tuncanbit/tvs/internal/infrastructure/rpc"
	"github.com/tuncanbit/tvs/internal/repositories/authrepo"
	"github.com/tuncanbit/tvs/internal/repositories/balancerepo"
	"github.com/tuncanbit/tvs/internal/repositories/sessionrepo"
	"github.com/tuncanbit/tvs/internal/repositories/transactionrepo"
	"github.com/tuncanbit/tvs/internal/repositories/withdrawalrepo"
	"github.com/tuncanbit/tvs/internal/server"
	"github.com/tuncanbit/tvs/internal/server/websocket"
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

	transactionRepo := transactionrepo.New(db, logger)
	sessionRepo := sessionrepo.New(db, logger)
	balanceRepo := balancerepo.New(db.Db)
	withdrawalRepo := withdrawalrepo.New(db.Db, logger)
	authRepo := authrepo.NewAuthRepository(db.Db)

	exchageApiClient := clients.NewExchangeAPIClient(&cfg.ExchangeAPIConfig, logger)
	heliusClient := rpc.NewHeliusClient(cfg, logger)
	wsHub := websocket.NewWsHub(logger)
	go wsHub.Run()

	verificationSvc := verificationservice.New(sessionRepo, transactionRepo, balanceRepo, withdrawalRepo, cfg.Verification, logger, heliusClient, exchageApiClient, wsHub)
	authSvc := authservice.NewAuthService(cfg, logger, authRepo)

	go verificationSvc.StartTransactionVerification(context.Background())

	srv := server.New(cfg, verificationSvc, authSvc, logger, wsHub)
	srv.Start()
}
