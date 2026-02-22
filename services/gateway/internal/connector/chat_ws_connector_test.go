package connector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChatWSConnector(t *testing.T) {
	connector := NewChatWSConnector("ws://localhost:8081/ws/chat")

	assert.NotNil(t, connector)
	assert.Equal(t, "ws://localhost:8081/ws/chat", connector.chatServiceURL)
	assert.NotNil(t, connector.connections)
	assert.Empty(t, connector.connections)
}

func TestChatWSConnector_Disconnect(t *testing.T) {
	connector := NewChatWSConnector("ws://localhost:8081/ws/chat")

	// Добавляем тестовое соединение
	connector.connections["test-user"] = &ChatConnection{
		userID: "test-user",
	}

	assert.Len(t, connector.connections, 1)

	// Disconnect
	connector.Disconnect("test-user")

	assert.Empty(t, connector.connections)
}

func TestChatWSConnector_Disconnect_NonExistent(t *testing.T) {
	connector := NewChatWSConnector("ws://localhost:8081/ws/chat")

	// Disconnect non-existent user should not panic
	assert.NotPanics(t, func() {
		connector.Disconnect("non-existent")
	})
}

func TestChatWSConnector_Close(t *testing.T) {
	connector := NewChatWSConnector("ws://localhost:8081/ws/chat")

	// Добавляем несколько тестовых соединений
	connector.connections["user1"] = &ChatConnection{userID: "user1"}
	connector.connections["user2"] = &ChatConnection{userID: "user2"}

	assert.Len(t, connector.connections, 2)

	// Close all
	connector.Close()

	assert.Empty(t, connector.connections)
}

func TestChatConnection_Fields(t *testing.T) {
	conn := &ChatConnection{
		userID: "test-user",
	}

	assert.Equal(t, "test-user", conn.userID)
	assert.Nil(t, conn.clientConn)
	assert.Nil(t, conn.chatServiceConn)
}

// Benchmark тесты
func BenchmarkNewChatWSConnector(b *testing.B) {
	for i := 0; i < b.N; i++ {
		connector := NewChatWSConnector("ws://localhost:8081/ws/chat")
		_ = connector
	}
}

func BenchmarkChatWSConnector_Disconnect(b *testing.B) {
	connector := NewChatWSConnector("ws://localhost:8081/ws/chat")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		userID := fmt.Sprintf("user-%d", i)
		connector.connections[userID] = &ChatConnection{userID: userID}
		b.StartTimer()

		connector.Disconnect(userID)
	}
}

func BenchmarkChatWSConnector_Close(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		connector := NewChatWSConnector("ws://localhost:8081/ws/chat")
		for j := 0; j < 100; j++ {
			userID := fmt.Sprintf("user-%d", j)
			connector.connections[userID] = &ChatConnection{userID: userID}
		}
		b.StartTimer()

		connector.Close()
	}
}
