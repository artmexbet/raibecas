# Chat Service Configuration Example

## Redis Configuration

The Redis store is now configured to persist chat history with the following parameters:

```yaml
redis:
  host: localhost
  port: "6379"
  db: 0
  # Chat history TTL in seconds (24 hours)
  chat_ttl: 86400
  # Temporary message buffer TTL in seconds (24 hours)
  message_ttl: 86400
```

## Environment Variables

You can also configure Redis using environment variables:

```bash
# Redis connection
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# TTL settings (in seconds)
REDIS_CHAT_TTL=86400        # 24 hours
REDIS_MESSAGE_TTL=86400     # 24 hours
```

## Docker Compose Example

```yaml
version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes

  chat-service:
    build:
      context: ..
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_DB: 0
      REDIS_CHAT_TTL: 86400
      REDIS_MESSAGE_TTL: 86400
      # Other settings...
    depends_on:
      - redis

volumes:
  redis_data:
```

## TTL Settings Guide

### Default TTL (24 hours = 86400 seconds)

- **Chat History**: Messages are kept for 24 hours
- **Temporary Messages**: Chunks are kept for 24 hours

### Recommended TTL Values

- **Short-lived (1 hour = 3600 seconds)**: For development/testing
- **Medium (8 hours = 28800 seconds)**: For typical usage
- **Long-lived (7 days = 604800 seconds)**: For important conversations
- **Permanent (0 = no expiration)**: Only set with external cleanup

## Monitoring Redis Storage

### Check current keys:

```bash
redis-cli
> KEYS "chat:*"
> GET "chat:history:user123"
> TTL "chat:history:user123"
```

### Monitor message chunks in real-time:

```bash
redis-cli
> MONITOR
```

### Calculate storage usage:

```bash
redis-cli
> DBSIZE                    # Total keys
> INFO memory              # Memory statistics
```

## Best Practices

1. **Set appropriate TTL**: Balance between storage and user experience
2. **Monitor memory usage**: Use `INFO memory` to track Redis usage
3. **Backup important chats**: Consider implementing periodic backups
4. **Use Redis persistence**: Enable AOF (Append-Only File) for durability
5. **Regular cleanup**: Set up cron jobs for manual cleanup if needed

## Performance Considerations

- **JSON serialization**: Messages are stored as JSON for compatibility
- **Key naming**: Keys use prefixes (`chat:history:`, `chat:temp_msg:`) for easy filtering
- **TTL efficiency**: Redis automatically removes expired keys
- **Multi-user support**: Each user has independent chat history

## API Endpoints

### POST /api/v1/chat
Send a message and get streaming response

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "input": "Your message here"
  }'
```

### DELETE /api/v1/chat/:userID
Clear chat history for a user

```bash
curl -X DELETE http://localhost:8080/api/v1/chat/user123
```

## Troubleshooting

### Issue: Messages not persisting

1. Check Redis connection: `redis-cli ping` should return PONG
2. Verify environment variables are set correctly
3. Check logs for warnings about Redis operations
4. Verify TTL is not set to 0

### Issue: Memory growing too fast

1. Check TTL settings - they might be too high
2. Monitor concurrent active chats
3. Analyze message sizes
4. Consider implementing manual cleanup procedures

### Issue: Accessing old messages fails

1. Check if TTL has expired: `TTL chat:history:userID`
2. Verify Redis has not been restarted (clears volatile data)
3. Check Redis persistence settings

