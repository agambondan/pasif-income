# Multi-Platform Distribution Design

This document outlines the architecture for a secure, scalable, and official multi-platform distribution system using OAuth-based account linking.

## 1. High-Level Concept
- **1 Video -> N Platforms -> M Accounts**.
- **OAuth-first**: No passwords stored. We use Official APIs (YouTube Data API, TikTok Business API, Meta Graph API).
- **Asynchronous Publishing**: Videos are generated first, then a specialized worker handles the distribution to all requested destinations.

---

## 2. Database Schema

### `connected_accounts`
Stores the OAuth credentials for each platform account.
| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID/TEXT (PK) | Unique account identifier (e.g., `yt-main`). |
| `platform` | TEXT | `youtube`, `tiktok`, `instagram`. |
| `display_name` | TEXT | Human readable name (e.g., "My Gaming Channel"). |
| `access_token` | TEXT (Encrypted) | Current valid OAuth access token. |
| `refresh_token`| TEXT (Encrypted) | Token used to get new access tokens. |
| `token_expiry` | TIMESTAMP | When the access token expires. |
| `user_id` | TEXT | Owner of this connection. |

### `distribution_jobs`
Tracks the status of each upload attempt per platform.
| Column | Type | Description |
|--------|------|-------------|
| `id` | SERIAL (PK) | |
| `generation_job_id` | TEXT (FK) | Link to the original content generation job. |
| `account_id` | TEXT (FK) | Link to `connected_accounts`. |
| `platform` | TEXT | Redundant but useful for filtering. |
| `status` | TEXT | `pending`, `uploading`, `completed`, `failed`. |
| `external_id` | TEXT | The ID of the post on the platform (e.g., YouTube Video ID). |
| `error` | TEXT | Error message if failed. |

---

## 3. OAuth Flow Implementation

1. **Initiation**: User clicks "Connect Account" -> Frontend redirects to Backend `/api/auth/{platform}`.
2. **Redirect**: Backend generates Auth URL with required scopes (e.g., `youtube.upload`) and redirects user to Platform login.
3. **Callback**: User approves -> Platform redirects back to `/api/auth/{platform}/callback` with a `code`.
4. **Exchange**: Backend exchanges `code` for `access_token` and `refresh_token`.
5. **Storage**: Tokens are encrypted using AES-256 (Master Key in Env) and saved to `connected_accounts`.

---

## 4. User Interface (Dashboard)

### Account Management
- A "Settings" or "Integrations" page listing all connected accounts.
- Status indicator (🟢 Connected / 🔴 Re-auth needed).
- Buttons to "Add Account" or "Disconnect".

### Job Creation Form
A dynamic checkbox list populated from `connected_accounts`:
```
[ ] YouTube
    [ ] yt-main
    [ ] yt-shorts-secondary
[ ] TikTok
    [ ] tt-brand
[ ] Instagram
    [ ] ig-main
```

---

## 5. Worker & Publisher Architecture

We follow the **Adapter Pattern** to handle different platform APIs through a unified interface.

### New Port: `Publisher`
```go
type Publisher interface {
    Publish(ctx context.Context, videoPath string, metadata Metadata, account ConnectedAccount) (externalID string, err error)
}
```

### Execution Flow
1. `GeneratorService` finishes video assembly and local storage.
2. It creates entries in `distribution_jobs` for every checked account.
3. A **Publisher Worker** (Go Routine or separate Service) picks up `pending` distribution jobs.
4. It fetches encrypted tokens, decrypts them, and calls the corresponding platform adapter.
5. If one platform fails, it updates the `distribution_jobs` error field but doesn't affect other platforms.

---

## 6. Implementation Payload Example

Request to `/api/generate`:
```json
{
  "niche": "stoicism",
  "topic": "how to control your mind",
  "destinations": [
    { "platform": "youtube", "account_id": "yt-main" },
    { "platform": "tiktok", "account_id": "tt-brand" }
  ]
}
```
