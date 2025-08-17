package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/tuncanbit/tvs/internal/application/services"
	"github.com/tuncanbit/tvs/internal/server/handlers"
	"github.com/tuncanbit/tvs/pkg/config"
)

type Server struct {
	VerificationSvc services.IVerificationService
	Cfg             *config.Config
	Logger          zerolog.Logger
	Router          *gin.Engine
	httpServer      *http.Server
}

func New(cfg *config.Config, verificationService services.IVerificationService, logger zerolog.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	return &Server{
		Cfg:             cfg,
		VerificationSvc: verificationService,
		Logger:          logger,
		Router:          router,
	}
}

func (s *Server) SetupRouter() {
	handler := handlers.New(
		s.VerificationSvc,
		s.Logger,
		s.Cfg,
	)
	handler.SetupHandlers(s.Router)
}

func (s *Server) Start() {
	s.SetupRouter()

	s.httpServer = &http.Server{
		Addr:         s.Cfg.Server.Host + ":" + s.Cfg.Server.Port,
		Handler:      s.Router,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	s.Logger.Info().Msgf("Starting server on %s", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	<-stopChan
	s.Logger.Info().Msg("Shutdown signal received, shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.Logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	s.Logger.Info().Msg("Server exited gracefully")
}
