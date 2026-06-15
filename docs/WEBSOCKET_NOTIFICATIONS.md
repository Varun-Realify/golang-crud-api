# WebSocket Notifications Implementation

## Overview

A real-time WebSocket notification system has been implemented to provide live updates on task status for Google Ads and Meta platforms. The system tracks task lifecycle events (initiated, processing, completed, failed) and broadcasts them to connected clients.

## Architecture

### Components

1. **Task Notification Model** (`models/task_notification.go`)
   - Stores task notification history in PostgreSQL
   - Tracks task status, progress, and error messages
   - Supports both Google and Meta platforms

2. **WebSocket Manager** (`websocket/manager.go`)
   - Manages individual client connections
   - Handles ping/pong keepalive
   - Provides connection lifecycle management

3. **WebSocket Hub** (`websocket/hub.go`)
   - Central message routing system
   - Maintains user-to-client mapping
   - Broadcasts messages to specific users or all users

4. **Notification Broadcaster** (`websocket/broadcaster.go`)
   - High-level API for emitting notifications
   - Saves notifications to database
   - Broadcasts via WebSocket and Redis pub/sub

5. **Notification Publisher** (`websocket/publisher.go`)
   - Redis-based pub/sub for cross-process communication
   - Enables worker process to publish notifications
   - Channels: `notifications:{platform}` and `notifications:user:{taskId}`

6. **WebSocket Handler** (`handlers/websocket_handler.go`)
   - HTTP endpoint for WebSocket upgrades
   - REST endpoints for notification history

## API Endpoints

### WebSocket Connection
```
GET /ws?user_id={user_id}
```

**Query Parameters:**
- `user_id` (required): UUID of the user

**Response:** WebSocket upgrade to persistent connection

**Example:**
```javascript
const userID = "123e4567-e89b-12d3-a456-426614174000";
const ws = new WebSocket(`ws://localhost:8080/ws?user_id=${userID}`);

ws.onmessage = (event) => {
  const notification = JSON.parse(event.data);
  console.log('Received notification:', notification);
};
```

### Notification History
```
GET /notifications/history?limit=50
```

**Query Parameters:**
- `limit` (optional): Number of notifications to retrieve (default: 50, max: 100)

**Response:**
```json
[
  {
    "id": "uuid",
    "task_id": "uuid",
    "task_type": "google:ads_ingest",
    "platform": "google",
    "status": "completed",
    "message": "Successfully synced 15 Google campaigns",
    "progress": 100,
    "created_at": "2024-01-15T10:30:00Z"
  }
]
```

### Get Notification by Task ID
```
GET /notifications?task_id={task_id}
```

**Query Parameters:**
- `task_id` (required): Task UUID

**Response:**
```json
{
  "id": "uuid",
  "task_id": "uuid",
  "task_type": "google:ads_ingest",
  "platform": "google",
  "status": "completed",
  "message": "Successfully synced 15 Google campaigns",
  "progress": 100,
  "details": "{}",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### WebSocket Stats
```
GET /notifications/stats
```

**Response:**
```json
{
  "total_clients": 5
}
```

## WebSocket Message Format

### Notification Message
```json
{
  "type": "notification",
  "task_id": "uuid",
  "task_type": "google:ads_ingest",
  "platform": "google",
  "status": "completed",
  "message": "Successfully synced 15 Google campaigns",
  "progress": 100,
  "timestamp": "2024-01-15T10:30:00Z",
  "details": {
    "total_campaigns": 15
  },
  "error_msg": ""
}
```

### Task Status Values
- `initiated` - Task has been queued
- `processing` - Task is being executed
- `completed` - Task completed successfully
- `failed` - Task failed with error
- `cancelled` - Task was cancelled
- `pending` - Task is pending

### Platform Values
- `google` - Google Ads
- `meta` - Meta Ads

## Task Types

### Google Ads
- `google:ads_ingest` - Sync Google campaigns from API
- `google:campaign_create` - Create a new campaign
- `google:adgroup_create` - Create a new ad group
- `google:ad_create` - Create a new ad

### Meta Ads
- `meta:ads_ingest` - Sync Meta campaigns from API
- `meta:campaign_create` - Create a new campaign
- `meta:adset_create` - Create a new ad set
- `meta:ad_create` - Create a new ad

## Updated Task Payloads

All task payloads now include a `user_id` field to enable targeted notifications:

```go
type GoogleAdsIngestPayload struct {
    UserID       string // NEW
    ResourceName string
    Type         string
}

type MetaCampaignCreatePayload struct {
    UserID      string // NEW
    AdAccountID string
    AccessToken string
    Req         models.CampaignCreate
}
// ... etc for all other payloads
```

## Usage Example

### 1. Client-Side (JavaScript)

```javascript
class NotificationManager {
  constructor(userID) {
    this.userID = userID;
    this.ws = null;
    this.listeners = [];
  }

  connect() {
    return new Promise((resolve, reject) => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const url = `${protocol}//${window.location.host}/ws?user_id=${this.userID}`;
      
      this.ws = new WebSocket(url);
      
      this.ws.onopen = () => {
        console.log('Connected to notifications');
        resolve();
      };
      
      this.ws.onmessage = (event) => {
        const notification = JSON.parse(event.data);
        this.emit('notification', notification);
      };
      
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        reject(error);
      };
      
      this.ws.onclose = () => {
        console.log('Disconnected from notifications');
        // Attempt to reconnect after 3 seconds
        setTimeout(() => this.connect(), 3000);
      };
    });
  }

  on(event, callback) {
    this.listeners.push({ event, callback });
  }

  emit(event, data) {
    this.listeners
      .filter(l => l.event === event)
      .forEach(l => l.callback(data));
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

// Usage
const notificationManager = new NotificationManager(userID);
await notificationManager.connect();

notificationManager.on('notification', (notification) => {
  console.log(`Task ${notification.task_id}: ${notification.status} - ${notification.message}`);
  
  if (notification.status === 'completed') {
    console.log('✓ Task completed!', notification.details);
  } else if (notification.status === 'failed') {
    console.error('✗ Task failed:', notification.error_msg);
  }
});
```

### 2. Server-Side (Go)

When enqueuing a task, include the user_id:

```go
// Example: Enqueueing a Google Ads sync task
payload, _ := json.Marshal(workers.GoogleAdsIngestPayload{
    UserID:       userID,
    ResourceName: "customers/1234567890",
    Type:         "campaign",
})

task := asynq.NewTask(workers.TypeGoogleAdsIngest, payload)
info, err := workers.EnqueueTask(task)
if err != nil {
    return err
}

// Notifications will be automatically sent as the task progresses
```

### 3. Fetching Notification History

```bash
# Get last 50 notifications
curl http://localhost:8080/notifications/history

# Get specific notification for a task
curl "http://localhost:8080/notifications?task_id=uuid"

# Get WebSocket stats
curl http://localhost:8080/notifications/stats
```

## Database Schema

### TaskNotification Table
```sql
CREATE TABLE task_notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  
  task_id VARCHAR(255) NOT NULL INDEX,
  task_type VARCHAR(255) NOT NULL,
  platform VARCHAR(50) NOT NULL,
  
  user_id UUID NOT NULL INDEX,
  
  status VARCHAR(50) NOT NULL INDEX,
  message TEXT,
  details TEXT,  -- JSON
  error_msg TEXT,
  progress INTEGER DEFAULT 0,
  
  metadata TEXT  -- JSON
);
```

## Environment Variables

No new environment variables required. Uses existing `REDIS_ADDR`:
- `REDIS_ADDR` - Redis server address (default: "127.0.0.1:6379")

## Implementation Details

### Worker Notifications

The worker process emits notifications through:

1. **Direct Database Saves** - TaskNotification records stored in PostgreSQL
2. **Redis Pub/Sub** - Notifications published to Redis channels
3. **WebSocket Broadcast** - Connected clients receive messages in real-time

### Notification Flow

```
Task Enqueued
    ↓
Worker Receives Task
    ↓
Broadcast "initiated"
    ↓
Broadcast "processing" (with progress updates)
    ↓
Task Execution
    ↓
On Success: Broadcast "completed" with details
On Failure: Broadcast "failed" with error message
    ↓
All Broadcasts:
  - Saved to database
  - Published to Redis pub/sub
  - Sent to WebSocket clients
```

### Scalability Considerations

1. **Multiple Worker Instances**: All workers publish to the same Redis instance, which routes to WebSocket clients
2. **Multiple API Instances**: WebSocket manager is per-API instance; use Redis pub/sub for cross-instance communication
3. **Client Reconnection**: Clients should implement exponential backoff when reconnecting
4. **Message Buffering**: Each client has a 256-message buffer; older messages are dropped if buffer overflows

## Monitoring & Debugging

### Check Active Connections
```bash
curl http://localhost:8080/notifications/stats
```

### View Recent Notifications
```bash
curl "http://localhost:8080/notifications/history?limit=20"
```

### Monitor Redis Channels
```bash
redis-cli
SUBSCRIBE notifications:google
SUBSCRIBE notifications:meta
SUBSCRIBE notifications:user:*
```

## Future Enhancements

1. **Message Filtering** - Allow clients to filter by task type or platform
2. **Persistent Queue** - Replay missed notifications if client reconnects
3. **Notification Preferences** - Per-user notification settings
4. **Batch Notifications** - Group similar notifications together
5. **Analytics** - Track notification delivery and client engagement
6. **Retry Logic** - Automatic retry for failed notifications
7. **Rate Limiting** - Prevent notification spam

## Testing

### Manual WebSocket Test

```bash
# Using websocat
websocat "ws://localhost:8080/ws?user_id=123e4567-e89b-12d3-a456-426614174000"

# Messages received will show as:
{"type":"notification","task_id":"uuid","task_type":"google:ads_ingest","status":"initiated",...}
```

### Unit Tests

See `websocket_test.go` for comprehensive unit tests covering:
- Client connection/disconnection
- Message broadcasting
- User isolation
- Connection limits
- Timeout handling
