# WebSocket Notifications Integration Guide

## Overview

This guide explains how to integrate the new WebSocket notification system into existing handlers and task enqueueing code.

## Key Changes

### 1. Task Payload Updates

All task payloads now require a `UserID` field to enable targeted notifications.

#### Before:
```go
workers.GoogleAdsIngestPayload{
    Type: "campaign",
}
```

#### After:
```go
user := handlers.GetUserContext(r)
workers.GoogleAdsIngestPayload{
    UserID: user.ID,
    Type:   "campaign",
}
```

## Handler Integration Examples

### Google Ads Sync Handler

**File:** `handlers/google_ads_handler.go`

```go
func SyncGoogleCampaigns(w http.ResponseWriter, r *http.Request) {
    // Get user context
    user := handlers.GetUserContext(r)
    
    // Enqueue task with user_id
    if err := workers.EnqueueTask(
        workers.TypeGoogleAdsIngest,
        workers.GoogleAdsIngestPayload{
            UserID: user.ID,
            Type:   "campaign",
        },
    ); err != nil {
        http.Error(w, "Failed to queue sync task: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Google sync task queued successfully",
        "status":  "processing",
    })
}
```

### Google Campaign Create Handler

```go
func CreateGoogleCampaign(w http.ResponseWriter, r *http.Request) {
    var req models.GoogleCampaignRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
        return
    }

    // Get user context
    user := handlers.GetUserContext(r)

    // Enqueue task with user_id
    if err := workers.EnqueueTask(
        workers.TypeGoogleCampaignCreate,
        workers.GoogleCampaignCreatePayload{
            UserID: user.ID,
            Req:    req,
        },
    ); err != nil {
        http.Error(w, "Failed to queue campaign creation: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Campaign creation task queued successfully",
        "status":  "processing",
    })
}
```

### Meta Campaign Create Handler

```go
func CreateCampaign(w http.ResponseWriter, r *http.Request) {
    var req models.CampaignCreate
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get user context
    user := handlers.GetUserContext(r)

    // Enqueue task with user_id
    if err := workers.EnqueueTask(
        workers.TypeMetaCampaignCreate,
        workers.MetaCampaignCreatePayload{
            UserID:      user.ID,
            AdAccountID: user.MetaAdAccountID,
            AccessToken: user.MetaAccessToken,
            Req:         req,
        },
    ); err != nil {
        http.Error(w, "Failed to queue campaign creation: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Meta campaign creation task queued successfully",
        "status":  "processing",
    })
}
```

### Meta AdSet Create Handler

```go
func CreateAdSet(w http.ResponseWriter, r *http.Request) {
    var req models.AdSetCreate
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get user context
    user := handlers.GetUserContext(r)

    // Enqueue task with user_id
    if err := workers.EnqueueTask(
        workers.TypeMetaAdSetCreate,
        workers.MetaAdSetCreatePayload{
            UserID:      user.ID,
            AdAccountID: user.MetaAdAccountID,
            AccessToken: user.MetaAccessToken,
            PixelID:     user.MetaPixelID,
            Req:         req,
        },
    ); err != nil {
        http.Error(w, "Failed to queue ad set creation: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Meta ad set creation task queued successfully",
        "status":  "processing",
    })
}
```

### Meta Ad Create Handler

```go
func CreateAd(w http.ResponseWriter, r *http.Request) {
    var req models.AdCreate
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get user context
    user := handlers.GetUserContext(r)

    // Enqueue task with user_id
    if err := workers.EnqueueTask(
        workers.TypeMetaAdCreate,
        workers.MetaAdCreatePayload{
            UserID:      user.ID,
            AdAccountID: user.MetaAdAccountID,
            AccessToken: user.MetaAccessToken,
            PageID:      user.MetaPageID,
            Req:         req,
        },
    ); err != nil {
        http.Error(w, "Failed to queue ad creation: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Meta ad creation task queued successfully",
        "status":  "processing",
    })
}
```

### Google AdGroup Create Handler

```go
func CreateGoogleAdGroup(w http.ResponseWriter, r *http.Request) {
    var req models.GoogleAdGroupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
        return
    }

    // Get user context
    user := handlers.GetUserContext(r)

    // Enqueue task with user_id
    if err := workers.EnqueueTask(
        workers.TypeGoogleAdGroupCreate,
        workers.GoogleAdGroupCreatePayload{
            UserID: user.ID,
            Req:    req,
        },
    ); err != nil {
        http.Error(w, "Failed to queue ad group creation: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Google ad group creation task queued successfully",
        "status":  "processing",
    })
}
```

### Google Ad Create Handler

```go
func CreateGoogleAd(w http.ResponseWriter, r *http.Request) {
    var req models.GoogleAdRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
        return
    }

    // Get user context
    user := handlers.GetUserContext(r)

    // Enqueue task with user_id
    if err := workers.EnqueueTask(
        workers.TypeGoogleAdCreate,
        workers.GoogleAdCreatePayload{
            UserID: user.ID,
            Req:    req,
        },
    ); err != nil {
        http.Error(w, "Failed to queue ad creation: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Google ad creation task queued successfully",
        "status":  "processing",
    })
}
```

### Meta Sync Handler

```go
func SyncCampaigns(w http.ResponseWriter, r *http.Request) {
    // Get user context
    user := handlers.GetUserContext(r)

    if err := workers.EnqueueTask(
        workers.TypeMetaAdsIngest,
        workers.MetaAdsIngestPayload{
            UserID:      user.ID,
            AdAccountID: user.MetaAdAccountID,
            AccessToken: user.MetaAccessToken,
            Type:        "campaign",
        },
    ); err != nil {
        http.Error(w, "Failed to queue sync task: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "Meta sync task queued successfully",
        "status":  "processing",
    })
}
```

## Client-Side Integration

### React Example

```jsx
import { useEffect, useState } from 'react';

function TaskNotifications({ userId }) {
  const [notifications, setNotifications] = useState([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws?user_id=${userId}`);

    ws.onopen = () => {
      console.log('Connected to notifications');
      setConnected(true);
    };

    ws.onmessage = (event) => {
      const notification = JSON.parse(event.data);
      setNotifications((prev) => [notification, ...prev.slice(0, 49)]);
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      setConnected(false);
    };

    ws.onclose = () => {
      setConnected(false);
      // Reconnect after 3 seconds
      setTimeout(() => {
        window.location.reload();
      }, 3000);
    };

    return () => ws.close();
  }, [userId]);

  return (
    <div className="notifications-container">
      <div className="connection-status">
        {connected ? '🟢 Connected' : '🔴 Disconnected'}
      </div>

      <div className="notifications-list">
        {notifications.map((notification) => (
          <div
            key={notification.task_id}
            className={`notification notification-${notification.status}`}
          >
            <div className="notification-header">
              <h4>{notification.task_type}</h4>
              <span className={`status ${notification.status}`}>
                {notification.status}
              </span>
            </div>
            <p>{notification.message}</p>
            <div className="progress-bar">
              <div
                className="progress"
                style={{ width: `${notification.progress}%` }}
              />
            </div>
            {notification.error_msg && (
              <p className="error">{notification.error_msg}</p>
            )}
            {notification.details && (
              <pre className="details">
                {JSON.stringify(notification.details, null, 2)}
              </pre>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

export default TaskNotifications;
```

## Migration Checklist

- [ ] Update all task payload structures to include `UserID` field
- [ ] Update all handler functions to extract user from context
- [ ] Update all `EnqueueTask` calls to include `UserID` in payloads
- [ ] Test WebSocket connection in frontend
- [ ] Verify notifications are being received in real-time
- [ ] Check database for TaskNotification records
- [ ] Monitor Redis pub/sub channels during testing
- [ ] Update API documentation with new notification endpoints
- [ ] Add notification UI components to frontend

## Verification Steps

1. **Check Database Migration**
   ```bash
   psql -c "SELECT * FROM task_notifications LIMIT 5;"
   ```

2. **Test WebSocket Connection**
   ```bash
   websocat "ws://localhost:8080/ws?user_id=<your-user-id>"
   ```

3. **Enqueue a Test Task**
   ```bash
   curl -X POST http://localhost:8080/google/sync \
     -H "X-User-Email: your@email.com"
   ```

4. **Check Notification History**
   ```bash
   curl http://localhost:8080/notifications/history
   ```

5. **Monitor Redis**
   ```bash
   redis-cli SUBSCRIBE "notifications:*"
   ```

## Troubleshooting

### Notifications not being received
1. Check if user_id is being passed correctly in task payloads
2. Verify WebSocket connection is established
3. Check Redis connection: `redis-cli ping`
4. Monitor worker logs for broadcast errors

### WebSocket connection fails
1. Ensure API server is running
2. Check CORS settings in main.go
3. Verify WebSocket handler is registered in routes

### Notifications not persisting to database
1. Check database connection
2. Verify TaskNotification migration ran
3. Check for database write errors in logs

## Next Steps

1. Update all remaining handlers to include UserID
2. Implement frontend UI for notifications
3. Add notification filtering and preferences
4. Implement notification persistence and replay
