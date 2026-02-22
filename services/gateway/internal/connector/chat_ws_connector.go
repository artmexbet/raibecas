package connector

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	fiberWs "github.com/gofiber/contrib/websocket"
)

// ChatWSConnector manages WebSocket connection to Chat service
type ChatWSConnector struct {
	chatServiceURL string
	connections    map[string]*ChatConnection
	mu             sync.RWMutex
}

// ChatConnection represents a single client connection through to chat service
type ChatConnection struct {
	clientConn      *fiberWs.Conn
	chatServiceConn *websocket.Conn
	userID          string
	cancel          context.CancelFunc
}

// NewChatWSConnector creates a new WebSocket connector to chat service
func NewChatWSConnector(chatServiceURL string) *ChatWSConnector {
	return &ChatWSConnector{
		chatServiceURL: chatServiceURL,
		connections:    make(map[string]*ChatConnection),
	}
}

// Connect establishes WebSocket connection to chat service and bridges it with client
func (c *ChatWSConnector) Connect(ctx context.Context, clientConn *fiberWs.Conn, userID string) error {
	// Connect to chat service via WebSocket using fasthttp/websocket
	url := fmt.Sprintf("%s?userID=%s", c.chatServiceURL, userID)

	// Create dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Add userID to headers
	headers := http.Header{}
	headers.Set("X-User-ID", userID)

	chatConn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to chat service: %w", err)
	}

	slog.Info("Connected to chat service WebSocket", "user_id", userID)

	ctx, cancel := context.WithCancel(ctx)

	conn := &ChatConnection{
		clientConn:      clientConn,
		chatServiceConn: chatConn,
		userID:          userID,
		cancel:          cancel,
	}

	c.mu.Lock()
	c.connections[userID] = conn
	c.mu.Unlock()

	// Start bidirectional forwarding
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Chat Service
	go func() {
		defer wg.Done()
		c.forwardClientToChat(ctx, conn)
	}()

	// Chat Service -> Client
	go func() {
		defer wg.Done()
		c.forwardChatToClient(ctx, conn)
	}()

	// Wait for either direction to close
	wg.Wait()

	// Cleanup
	c.Disconnect(userID)

	return nil
}

// forwardClientToChat forwards messages from client to chat service
func (c *ChatWSConnector) forwardClientToChat(ctx context.Context, conn *ChatConnection) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgType, data, err := conn.clientConn.ReadMessage()
		if err != nil {
			slog.Debug("Client WebSocket closed", "user_id", conn.userID, "error", err)
			return
		}

		if err := conn.chatServiceConn.WriteMessage(msgType, data); err != nil {
			slog.Error("Failed to forward to chat service", "user_id", conn.userID, "error", err)
			return
		}
	}
}

// forwardChatToClient forwards messages from chat service to client
func (c *ChatWSConnector) forwardChatToClient(ctx context.Context, conn *ChatConnection) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgType, data, err := conn.chatServiceConn.ReadMessage()
		if err != nil {
			slog.Debug("Chat service WebSocket closed", "user_id", conn.userID, "error", err)
			return
		}

		if err := conn.clientConn.WriteMessage(msgType, data); err != nil {
			slog.Error("Failed to forward to client", "user_id", conn.userID, "error", err)
			return
		}
	}
}

// Disconnect closes connection for specific user
func (c *ChatWSConnector) Disconnect(userID string) {
	c.mu.Lock()
	conn, exists := c.connections[userID]
	if exists {
		delete(c.connections, userID)
	}
	c.mu.Unlock()

	if conn != nil {
		if conn.cancel != nil {
			conn.cancel()
		}
		if conn.chatServiceConn != nil {
			conn.chatServiceConn.Close() //nolint:errcheck // safe to ignore
		}
		slog.Info("Disconnected from chat service", "user_id", userID)
	}
}

// Close closes all connections
func (c *ChatWSConnector) Close() {
	c.mu.Lock()
	connections := make([]*ChatConnection, 0, len(c.connections))
	for _, conn := range c.connections {
		connections = append(connections, conn)
	}
	c.connections = make(map[string]*ChatConnection)
	c.mu.Unlock()

	for _, conn := range connections {
		if conn.cancel != nil {
			conn.cancel()
		}
		if conn.chatServiceConn != nil {
			conn.chatServiceConn.Close() //nolint:errcheck // safe to ignore
		}
	}
}
