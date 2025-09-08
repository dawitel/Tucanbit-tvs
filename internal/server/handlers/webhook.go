package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/application/verificationservice"
)

type WebhookHandler struct {
	verificationSvc verificationservice.IVerificationService
	logger          zerolog.Logger
}

func NewWebhookHandler(verificationSvc verificationservice.IVerificationService, logger zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{
		verificationSvc: verificationSvc,
		logger:          logger,
	}
}

func (h *WebhookHandler) HandlePDMWebhook(c *gin.Context) {

}
