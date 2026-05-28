# Realify Ads Engine

A high-performance Marketing API Engine built with Go, focusing on multi-platform ad management for **Meta Ads** and **Google Ads**.

## Features

- **Meta Ads Integration (Multi-User)**:
  - Full CRUD for Campaigns, Ad Sets, and Ads.
  - Manual Background Sync: Fetch latest campaign data and status on-demand.
  - Real-time Insights and Stats.
  - Multi-tenant architecture — each user's credentials, ad account, and campaigns are fully isolated.
  - One-step onboarding: exchange a short-lived token and save credentials in a single API call.

- **Google Ads Integration (Single-User)**:
  - OAuth2 Flow implementation (Authorization Code Grant).
  - Manual Background Sync: Fetch latest campaign data and status on-demand.
  - GAQL (Google Ads Query Language) for listing campaigns.
  - Automated "Create-as-Enabled" workflow for instant deployment.

- **Async Ingestion via Asynq + Redis**:
  - All create operations and sync operations are processed as background tasks — the API returns `202 Accepted` immediately.
  - The worker calls the external platform API, performs an **Upsert** into PostgreSQL, then queries the DB to verify and log the ingestion process.
  - Failed tasks are automatically retried by Asynq.

- **High-Speed Querying**:
  - All listing endpoints (GET) read from the local PostgreSQL database, not the live API (~90% lower latency).
  - Use the `/sync` endpoints to refresh your local data from live platform APIs.
  - Insights/performance endpoints always hit the live API.

## Architecture

Two processes run side-by-side:

```
API Server (cmd/api)          Worker (cmd/worker)
      │                              │
      │  POST /meta/sync             │
      │──► enqueue task ──► Redis ──►│
      │    202 Accepted              │  1. Call Meta/Google API
      │                              │  2. Upsert results to PostgreSQL
      │                              │  3. Verify ingestion via DB query
```

## Quick Start: Google Ads Flow

Deploy your ads in 4 steps (each returns 202 — the worker handles the rest):

1. `POST /google/campaigns`
2. `POST /google/adgroups`
3. `POST /google/keywords`
4. `POST /google/ads`

## Meta Multi-User Flow

Each Meta user onboards once, then uses a header on all requests.

### Step 1 — Onboard (once per user)

Get a short-lived token from the [Meta Graph API Explorer](https://developers.facebook.com/tools/explorer), then call:

```bash
POST /meta/auth/token
Content-Type: application/json

{
  "short_token":   "<short-lived token from Graph API Explorer>",
  "email":         "user@example.com",
  "ad_account_id": "act_XXXXXXXXXX",
  "page_id":       "XXXXXXXXXX",
  "pixel_id":      "XXXXXXXXXX"
}
```

The server exchanges the token for a 60-day long-lived token, resolves the Meta user ID, and saves the user row in the local database.

### Step 2 — All subsequent requests

Pass the `X-User-Email` header. The server looks up that user's credentials automatically:

```bash
GET  /meta/campaigns         -H "X-User-Email: user@example.com"
POST /meta/campaigns         -H "X-User-Email: user@example.com"
GET  /meta/ads/{id}/insights -H "X-User-Email: user@example.com"
```

Omitting the header falls back to the system default credentials in `.env`.

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.20+ |
| Database | PostgreSQL + GORM |
| Router | Gorilla Mux |
| Task Queue | Asynq + Redis |
| API Docs | Swaggo (Swagger 2.0) |

## Project Structure

```
├── cmd/
│   ├── api/        # HTTP server entry point
│   └── worker/     # Background task worker entry point
├── config/         # Environment & OAuth token exchange
├── database/       # DB connection & AutoMigrate
├── docs/           # Auto-generated Swagger docs
├── handlers/       # HTTP controllers (enqueue tasks for writes)
├── models/         # GORM models + API request structs
├── services/       # Platform API calls + DB ingestion
├── workers/        # Asynq client, task type constants, payload structs
└── public/         # Frontend static files
```

## Setup & Installation

### 1. Clone the repository

```bash
git clone https://github.com/Varun-Realify/golang-crud-api.git
cd golang-crud-api
```

### 2. Configure environment

Create a `.env` file in the project root:

```env
SERVER_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=root
DB_NAME=Realify
DB_SSLMODE=disable

# Meta Ads
META_APP_ID=
META_APP_SECRET=
META_ACCESS_TOKEN=        # Default fallback (used when no X-User-Email header)
META_AD_ACCOUNT_ID=       # Default fallback ad account
META_PAGE_ID=
META_CATALOG_ID=
META_PIXEL_ID=

# Google Ads
GOOGLE_ADS_CUSTOMER_ID=
GOOGLE_ADS_LOGIN_CUSTOMER_ID=   # Only needed for MCC/manager accounts
GOOGLE_ADS_DEVELOPER_TOKEN=
GOOGLE_ADS_CLIENT_ID=
GOOGLE_ADS_CLIENT_SECRET=
GOOGLE_ADS_REFRESH_TOKEN=       # Obtained after completing the OAuth flow
GOOGLE_ADS_REDIRECT_URL=http://localhost:8080/callback

# Background worker
REDIS_ADDR=127.0.0.1:6379
```

### 3. Install dependencies

```bash
go mod tidy
```

### 4. Start Redis

Redis must be running before starting the worker. Default address: `127.0.0.1:6379`.

### 5. Run the API server

```bash
# Recommended: build binary first to avoid go run cache issues
go build -o realify-api ./cmd/api/... && ./realify-api
```

### 6. Run the background worker (separate terminal)

```bash
go run cmd/worker/main.go
```

The worker connects to the same Redis instance and processes all queued create tasks.

## API Documentation

Swagger UI: `http://localhost:8080/swagger/index.html`

> **Note:** The `X-User-Email` header appears in Swagger's parameter fields for all Meta endpoints.

### Meta Ads Endpoints

| Method | Endpoint | Description | Response |
| :--- | :--- | :--- | :--- |
| POST | `/meta/auth/token` | Onboard user: exchange short-lived token, save credentials | 200 |
| GET | `/meta/status` | Check Meta API connection and list pages/catalogs | 200 |
| GET | `/meta/campaigns` | List campaigns (from local DB) | 200 |
| POST | `/meta/sync` | Queue background campaign sync from Meta | **202** |
| POST | `/meta/campaigns` | Queue campaign creation | **202** |
| PATCH | `/meta/campaigns/{id}` | Update a campaign | 200 |
| POST | `/meta/adsets` | Queue ad set creation | **202** |
| GET | `/meta/adsets/{id}/ads` | List ads within an ad set (from local DB) | 200 |
| POST | `/meta/ads` | Queue ad creation | **202** |
| GET | `/meta/ads/{id}/insights` | Live ad performance metrics | 200 |
| GET | `/meta/campaigns/{id}/insights` | Live campaign performance metrics | 200 |

### Google Ads Endpoints

| Method | Endpoint | Description | Response |
| :--- | :--- | :--- | :--- |
| GET | `/google/auth` | Start OAuth2 flow (returns login URL) | 200 |
| GET | `/callback` | OAuth2 callback — exchanges code for tokens | 200 |
| GET | `/google/customers` | List accessible Google Ads accounts | 200 |
| GET | `/google/campaigns` | List campaigns (from local DB) | 200 |
| POST | `/google/sync` | Queue background campaign sync from Google | **202** |
| POST | `/google/campaigns` | Queue campaign creation | **202** |
| DELETE | `/google/campaigns` | Delete a campaign | 200 |
| GET | `/google/campaigns/{id}/insights` | Live campaign performance metrics | 200 |
| GET | `/google/adgroups` | List ad groups (from local DB) | 200 |
| POST | `/google/adgroups` | Queue ad group creation | **202** |
| GET | `/google/adgroups/{id}/insights` | Live ad group performance metrics | 200 |
| GET | `/google/ads` | List ads (from local DB) | 200 |
| POST | `/google/ads` | Queue responsive search ad creation | **202** |
| POST | `/google/keywords` | Add keywords to an ad group | 200 |
| GET | `/google/keywords` | List keywords for an ad group | 200 |
| DELETE | `/google/keywords` | Remove a keyword | 200 |

## Google OAuth Flow

1. `GET /google/auth` → copy the `login_url` from the response.
2. Paste into a browser and approve permissions.
3. Google redirects to `/callback` — copy the `refresh_token` from the response.
4. Save it to `GOOGLE_ADS_REFRESH_TOKEN` in `.env`.