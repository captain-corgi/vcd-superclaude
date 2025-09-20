package trpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anh-tuan-l/multi-agent-system/internal/agents"
	"go.uber.org/zap"
)

type Client struct {
	agents  map[string]string
	mu      sync.RWMutex
	logger  *zap.Logger
	timeout time.Duration
	retries int
}

type Message struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Content     interface{}       `json:"content"`
	FromAgent   string            `json:"from_agent"`
	ToAgent     string            `json:"to_agent"`
	Timestamp   time.Time         `json:"timestamp"`
	Correlation string            `json:"correlation_id"`
	Metadata    map[string]string `json:"metadata"`
}

type Response struct {
	ID        string      `json:"id"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Error     string      `json:"error,omitempty"`
	FromAgent string      `json:"from_agent"`
	Timestamp time.Time   `json:"timestamp"`
}

func NewClient(logger *zap.Logger, timeout time.Duration, retries int) *Client {
	return &Client{
		agents:  make(map[string]string),
		logger:  logger,
		timeout: timeout,
		retries: retries,
	}
}

func (c *Client) RegisterAgent(agentID, address string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.agents[agentID] = address
	c.logger.Info("Agent registered",
		zap.String("agent_id", agentID),
		zap.String("address", address))
}

func (c *Client) SendMessage(ctx context.Context, message *agents.Message) (*agents.Response, error) {
	c.mu.RLock()
	targetAddress, exists := c.agents[message.ToAgent]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s not found", message.ToAgent)
	}

	var lastErr error

	for i := 0; i < c.retries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Simulate sending message to target agent
			response, err := c.sendToAgent(ctx, targetAddress, message)
			if err == nil {
				return response, nil
			}

			lastErr = err
			c.logger.Warn("Message send failed, retrying",
				zap.String("from_agent", message.FromAgent),
				zap.String("to_agent", message.ToAgent),
				zap.Int("attempt", i+1),
				zap.Error(err))

			// Wait before retry
			if i < c.retries-1 {
				select {
				case <-time.After(time.Second * time.Duration(i+1)):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
		}
	}

	return nil, fmt.Errorf("failed to send message after %d attempts: %w", c.retries, lastErr)
}

func (c *Client) sendToAgent(ctx context.Context, address string, message *agents.Message) (*agents.Response, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// In a real implementation, this would make an HTTP/gRPC call
	// For now, we'll simulate the communication

	c.logger.Debug("Sending message to agent",
		zap.String("address", address),
		zap.String("message_type", string(message.Type)),
		zap.String("from_agent", message.FromAgent),
		zap.String("to_agent", message.ToAgent))

	// Simulate processing time
	select {
	case <-time.After(100 * time.Millisecond):
		// Simulate successful response
		return &agents.Response{
			ID:          fmt.Sprintf("resp_%d", time.Now().Unix()),
			Success:     true,
			Data:        map[string]interface{}{"status": "processed", "plan_id": message.Correlation},
			TargetAgent: message.FromAgent,
			Timestamp:   time.Now(),
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) GetAgentStatus(agentID string) (string, error) {
	c.mu.RLock()
	_, exists := c.agents[agentID]
	c.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("agent %s not found", agentID)
	}

	return "active", nil
}

func (c *Client) ListAgents() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	agents := make([]string, 0, len(c.agents))
	for id := range c.agents {
		agents = append(agents, id)
	}

	return agents
}

func (c *Client) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"registered_agents": len(c.agents),
		"timeout":           c.timeout.String(),
		"retries":           c.retries,
		"timestamp":         time.Now(),
	}
}

// Broadcast message to all registered agents
func (c *Client) BroadcastMessage(ctx context.Context, message *agents.Message) ([]*agents.Response, error) {
	c.mu.RLock()
	targetAgents := make([]string, 0, len(c.agents))
	for agentID := range c.agents {
		if agentID != message.FromAgent { // Don't send back to sender
			targetAgents = append(targetAgents, agentID)
		}
	}
	c.mu.RUnlock()

	responses := make([]*agents.Response, 0, len(targetAgents))
	var responseMutex sync.Mutex
	var wg sync.WaitGroup

	for _, agentID := range targetAgents {
		wg.Add(1)
		go func(targetAgent string) {
			defer wg.Done()

			// Create a message for this specific agent
			agentMessage := &agents.Message{
				ID:          fmt.Sprintf("msg_%d_%s", time.Now().Unix(), targetAgent),
				Type:        message.Type,
				Content:     message.Content,
				FromAgent:   message.FromAgent,
				ToAgent:     targetAgent,
				Timestamp:   time.Now(),
				Correlation: message.Correlation,
			}

			// Send message to this agent
			resp, err := c.SendMessage(ctx, agentMessage)
			if err != nil {
				c.logger.Error("Failed to broadcast message to agent",
					zap.String("target_agent", targetAgent),
					zap.Error(err))
				return
			}

			responseMutex.Lock()
			responses = append(responses, resp)
			responseMutex.Unlock()
		}(agentID)
	}

	wg.Wait()

	return responses, nil
}
