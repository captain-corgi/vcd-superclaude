package trpc

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/internal/agents"
)

// ValidateServiceToken validates that the service token in the message metadata matches the expected token
func ValidateServiceToken(message *agents.Message, expectedToken string, logger *zap.Logger) error {
	if expectedToken == "" {
		logger.Debug("No service token configured, skipping validation")
		return nil
	}

	if message.Metadata == nil {
		logger.Warn("Message missing metadata for service token validation",
			zap.String("from_agent", message.FromAgent),
			zap.String("to_agent", message.ToAgent))
		return fmt.Errorf("missing message metadata")
	}

	actualToken, exists := message.Metadata["X-Service-Token"]
	if !exists {
		logger.Warn("Message missing service token",
			zap.String("from_agent", message.FromAgent),
			zap.String("to_agent", message.ToAgent))
		return fmt.Errorf("missing service token")
	}

	if actualToken != expectedToken {
		logger.Warn("Invalid service token",
			zap.String("from_agent", message.FromAgent),
			zap.String("to_agent", message.ToAgent))
		return fmt.Errorf("invalid service token")
	}

	logger.Debug("Service token validated successfully",
		zap.String("from_agent", message.FromAgent),
		zap.String("to_agent", message.ToAgent))

	return nil
}

// RequireServiceTokenMiddleware creates middleware that validates service tokens for A2A communication
func RequireServiceTokenMiddleware(expectedToken string, logger *zap.Logger) func(*agents.Message) error {
	return func(message *agents.Message) error {
		return ValidateServiceToken(message, expectedToken, logger)
	}
}
