package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	authservice "github.com/tuncanbit/tvs/internal/application/auth"
	"github.com/tuncanbit/tvs/internal/application/verificationservice"
	"github.com/tuncanbit/tvs/internal/server/middleware"
	"github.com/tuncanbit/tvs/internal/server/websocket"
	"github.com/tuncanbit/tvs/pkg/config"
)

type Handlers struct {
	VerificationSvc verificationservice.IVerificationService
	AuthSvc         authservice.IAuthService
	Logger          zerolog.Logger
	Config          *config.Config
	WsHub           *websocket.WsHub
}

func New(verificationSvc verificationservice.IVerificationService, AuthSvc authservice.IAuthService, logger zerolog.Logger, config *config.Config, wsHub *websocket.WsHub) *Handlers {
	return &Handlers{
		VerificationSvc: verificationSvc,
		AuthSvc:         AuthSvc,
		Logger:          logger,
		Config:          config,
		WsHub:           wsHub,
	}
}

func (h *Handlers) SetupHandlers(router *gin.Engine) {
	m := middleware.NewMiddleware(h.AuthSvc, h.Logger)
	m.SetupMiddleware(router)

	healthHandler := NewHealthHandler()
	sessionStatusHandler := NewSessionStatusHandler(h.WsHub, h.Logger)
	webhookHandler := NewWebhookHandler(h.VerificationSvc, h.Logger)
	messageHandler := NewMessageHandler(h.WsHub)

	router.GET("/health", healthHandler.Health)

	esRoute := router.Group("/tvs/api/es").Use(m.APIKeyMiddleware())
	{
		esRoute.POST("/messages/send", messageHandler.HandleMessage)
	}

	v1 := router.Group("/tvs/api/v1").Use(m.AuthMiddleware())
	{
		v1.GET("/status/ws", sessionStatusHandler.HandleWebSocket)
		v1.GET("/webhook/verify", webhookHandler.HandlePDMWebhook)
		v1.GET("/test", sessionStatusHandler.TestWebSocket)
	}
}
