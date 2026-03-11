# Chat Persistence Usage Examples

## Architecture Overview

```
User Request
     ↓
┌─────────────────────────────┐
│   HTTP Handler              │
│ POST /api/v1/chat           │
└──────────────┬──────────────┘
               ↓
┌─────────────────────────────┐
│   Chat Service              │
│ ProcessInput()              │
└──────────────┬──────────────┘
               ↓
    ┌──────────┴──────────┐
    ↓                     ↓
┌─────────────┐    ┌──────────────┐
│ Neuro Conn  │    │ Vector Store │
│ (Embeddings)│    │ (Qdrant)     │
└──────┬──────┘    └──────────────┘
       ↓
┌──────────────────────────────┐
│  Generate Response (chunks)  │
└──────────────┬───────────────┘
               ↓
┌──────────────────────────────────────┐
│  Redis Store                         │
│  • Save user message                 │
│  • Buffer response chunks            │
│  • Save complete assistant message   │
└──────────────┬───────────────────────┘
               ↓
        Stream to Client
```

## Detailed Flow: Single Message Processing

```
1. Client sends: {"user_id": "alice", "input": "What is AI?"}

2. Service retrieves history from Redis
   Key: chat:history:alice
   Value: [{role: "user", content: "Hi"}, {role: "assistant", content: "Hello"}]
   └─ Used for context in next request

3. Service saves user message to history
   Key: chat:history:alice
   New value: [...previous messages, {role: "user", content: "What is AI?"}]
   TTL: 24 hours

4. Service generates embeddings and retrieves documents

5. Service sends request to neuro (with history as context)

6. Neuro streams response in chunks:
   
   Chunk 1: "AI stands for"
   └─ Save to: chat:temp_msg:alice = "AI stands for"
   └─ Stream to client
   
   Chunk 2: " Artificial"
   └─ Update: chat:temp_msg:alice = "AI stands for Artificial"
   └─ Stream to client
   
   Chunk 3: " Intelligence"
   └─ Update: chat:temp_msg:alice = "AI stands for Artificial Intelligence"
   └─ Stream to client
   
   Chunk 4: " (complete)"
   └─ Update: chat:temp_msg:alice = "AI stands for Artificial Intelligence"
   └─ Save to history: chat:history:alice
   └─ Delete: chat:temp_msg:alice
   └─ Stream to client

7. Response complete, connection closes
```

## Example Scenarios

### Scenario 1: New User Starting Chat

```
User: user_new_001
Action: POST /api/v1/chat

Request:
{
  "user_id": "user_new_001",
  "input": "Hello, how are you?"
}

Execution:
1. Retrieve history: chat:history:user_new_001
   → No key found, empty history []
   
2. Save user message:
   Key: chat:history:user_new_001
   Value: [{"role": "user", "content": "Hello, how are you?"}]
   TTL: 86400 seconds

3. Get response from neuro in chunks
   
4. Save complete assistant response:
   Key: chat:history:user_new_001
   Value: [
     {"role": "user", "content": "Hello, how are you?"},
     {"role": "assistant", "content": "I'm doing great, thanks for asking!"}
   ]
   TTL: 86400 seconds

Final Redis state:
chat:history:user_new_001 = [user message, assistant message]
chat:temp_msg:user_new_001 = deleted
```

### Scenario 2: Continuing Conversation

```
User: alice (has existing chat)
Action: POST /api/v1/chat

Request:
{
  "user_id": "alice",
  "input": "Tell me more about machine learning"
}

Execution:
1. Retrieve history: chat:history:alice
   → Found: [
       {"role": "user", "content": "What is AI?"},
       {"role": "assistant", "content": "AI is artificial intelligence..."},
       {"role": "user", "content": "What about ML?"},
       {"role": "assistant", "content": "ML is machine learning..."}
     ]

2. Save new user message:
   Append to existing history in chat:history:alice
   → Now has 5 messages (4 + new user message)

3. Send to neuro WITH history as context
   → Neuro can reference previous conversation

4. Save assistant's response:
   Append to history
   → Now has 6 messages (5 + new assistant message)

Redis state after:
chat:history:alice = [
  {user: "What is AI?"},
  {asst: "AI is artificial..."},
  {user: "What about ML?"},
  {asst: "ML is machine..."},
  {user: "Tell me more about machine learning"},
  {asst: "Machine learning is..."}
]
All messages visible in next request
```

### Scenario 3: Clear Chat History

```
User: bob
Action: DELETE /api/v1/chat/bob

Execution:
1. Delete: chat:history:bob
2. Delete: chat:temp_msg:bob

Before:
  chat:history:bob = [many messages]
  chat:temp_msg:bob = "incomplete message..."

After:
  (both keys deleted)

Next POST /api/v1/chat for bob:
  → Will start with empty history
  → Like a completely new conversation
```

### Scenario 4: Handling Network Interruption

```
User: charlie
In-flight: Receiving response chunk 3 of 5

Scenario A: Network interrupts BEFORE completion
  chat:history:charlie = [previous messages]
  chat:temp_msg:charlie = "chunk 1... chunk 2... chunk 3..." (incomplete)
  
  When client reconnects and sends new message:
  1. History is still available (previous messages)
  2. Incomplete response is still in temp buffer
  3. New message is processed with full history
  → User doesn't lose conversation context

Scenario B: Network interrupts AFTER completion
  chat:history:charlie = [..., complete new message]
  chat:temp_msg:charlie = deleted
  
  When client reconnects:
  1. Full message already saved
  2. Can continue conversation normally
```

### Scenario 5: TTL Expiration

```
Message created at: 2025-01-15 10:00:00
TTL: 86400 seconds (24 hours)
Expiration time: 2025-01-16 10:00:00

During first 24 hours:
  chat:history:userID = [messages]  ← Available

After 24 hours:
  chat:history:userID = deleted automatically by Redis
  
Next request from user:
  1. Retrieve history → No key found
  2. History = []
  3. Starts new conversation

Note: MessageTTL is same as ChatTTL, so temp chunks
expire at same time as chat history.
```

## Code Examples

### Direct Redis Operations (for debugging)

```bash
# Check user's chat history
redis-cli GET "chat:history:alice"

# Check user's temporary message buffer
redis-cli GET "chat:temp_msg:alice"

# Check TTL (time until expiration)
redis-cli TTL "chat:history:alice"

# Clear a user's chat
redis-cli DEL "chat:history:alice" "chat:temp_msg:alice"

# Find all chats for monitoring
redis-cli KEYS "chat:history:*"

# Get chat size
redis-cli LLEN "chat:history:alice"  # Note: stored as string, not list
```

### Service Integration Example

```go
// In your application initialization
import (
    "github.com/artmexbet/raibecas/services/chat/internal/redis"
    "github.com/artmexbet/raibecas/services/chat/internal/service"
    "github.com/redis/go-redis/v9"
)

func setupChat(cfg *config.Config) (*service.Chat, error) {
    // Setup Redis
    redisClient := redis.NewClient(&redis.Options{
        Addr: cfg.Redis.GetAddress(),
        DB:   cfg.Redis.DB,
    })
    
    redisStore := redis.New(&cfg.Redis, redisClient)
    
    // Setup other components
    vectorStore := setupVectorStore(cfg)
    neuroConn := setupNeuro(cfg)
    
    // Create service with Redis store
    chatService := service.New(vectorStore, neuroConn, redisStore)
    
    return chatService, nil
}
```

## Testing Scenarios

### Unit Test: Message Persistence

```go
func TestMessagePersistence(t *testing.T) {
    r := setupRedis()
    ctx := context.Background()
    userID := "test_user"
    
    // Save message
    msg := domain.Message{Role: "user", Content: "Hello"}
    r.SaveMessage(ctx, userID, msg)
    
    // Retrieve and verify
    history, _ := r.RetrieveChatHistory(ctx, userID)
    assert.Len(t, history, 1)
    assert.Equal(t, "Hello", history[0].Content)
}
```

### Integration Test: Full Chat Flow

```go
func TestFullChatFlow(t *testing.T) {
    r := setupRedis()
    ctx := context.Background()
    userID := "alice"
    
    // Simulate incoming message
    userMsg := domain.Message{Role: "user", Content: "Hi"}
    r.SaveMessage(ctx, userID, userMsg)
    
    // Simulate streaming response
    r.AppendMessageChunk(ctx, userID, "Hello ", false)
    r.AppendMessageChunk(ctx, userID, "there", false)
    complete, _ := r.AppendMessageChunk(ctx, userID, "!", true)
    
    // Save complete message
    assistantMsg := domain.Message{Role: "assistant", Content: complete}
    r.SaveMessage(ctx, userID, assistantMsg)
    
    // Verify final state
    history, _ := r.RetrieveChatHistory(ctx, userID)
    assert.Len(t, history, 2)
    assert.Equal(t, "Hello there!", history[1].Content)
}
```

## Monitoring and Maintenance

### Regular Health Checks

```bash
# Check Redis connectivity
redis-cli ping

# Monitor active chats
redis-cli KEYS "chat:history:*" | wc -l

# Monitor storage usage
redis-cli INFO memory

# Check for orphaned temp messages
redis-cli KEYS "chat:temp_msg:*"
```

### Cleanup Operations

```bash
# Clear all chats for testing
redis-cli FLUSHDB

# Remove specific user's chat
redis-cli DEL "chat:history:user123" "chat:temp_msg:user123"

# Find and list old chats (without accessing content)
redis-cli KEYS "chat:history:*"
```

## Troubleshooting Common Issues

### Issue: Very large response messages
**Solution**: Adjust MessageTTL to clean up faster, or implement message size limits

### Issue: Storage growing quickly
**Solution**: Reduce ChatTTL, implement periodic cleanup, or archive old chats

### Issue: Users report incomplete messages
**Solution**: Check Redis availability, verify AppendMessageChunk completion flow

### Issue: Mixed up user conversations
**Solution**: Verify userID is properly passed through all layers, check Redis key prefixes

