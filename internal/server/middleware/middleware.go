package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	authservice "github.com/tuncanbit/tvs/internal/application/auth"
)

type Middleware struct {
	AuthSvc authservice.IAuthService
	logger  zerolog.Logger
}

func NewMiddleware(AuthSvc authservice.IAuthService, logger zerolog.Logger) *Middleware {
	return &Middleware{
		logger:  logger,
		AuthSvc: AuthSvc,
	}
}

func (m *Middleware) SetupMiddleware(router *gin.Engine) {
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})

	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		log.Info().
			Str("method", param.Method).
			Str("path", param.Path).
			Int("status", param.StatusCode).
			Dur("latency", param.Latency).
			Str("client_ip", param.ClientIP).
			Str("user_agent", param.Request.UserAgent()).
			Msg("HTTP Request")
		return ""
	}))

	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	})
}

func (m *Middleware) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				m.logger.Error().Msg("Invalid Authorization header format")
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid Authorization header format, expected 'Bearer <token>'",
				})
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else {
			tokenString = c.Query("token")
			if tokenString == "" {
				m.logger.Error().Msg("Authorization token missing")
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Authorization token required via Authorization header or token query parameter",
				})
				c.Abort()
				return
			}
		}

		claims, err := m.AuthSvc.VerifyToken(c.Request.Context(), tokenString)
		if err != nil {
			m.logger.Error().Err(err).Str("token", tokenString).Msg("Failed to verify token")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID.String())
		c.Set("is_verified", claims.IsVerified)
		c.Set("email_verified", claims.EmailVerified)
		c.Set("phone_verified", claims.PhoneVerified)

		c.Next()
	}
}

func (m *Middleware) APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var apiKey string

		apiKeyHeader := c.GetHeader("X-API-Key")
		if apiKeyHeader != "" {
			apiKey = apiKeyHeader
		} else {
			apiKey = c.Query("api_key")
			if apiKey == "" {
				m.logger.Error().Msg("API key missing")
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "API key required via X-API-Key header or api_key query parameter",
				})
				c.Abort()
				return
			}
		}

		if err := m.AuthSvc.VerifyAPIKey(c.Request.Context(), apiKey); err != nil {
			m.logger.Error().Err(err).Str("api_key", apiKey).Msg("Failed to verify API key")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
