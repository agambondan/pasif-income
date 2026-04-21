# Multi-Platform Distribution Design

This document outlines the architecture for a multi-platform distribution system with two integration paths:

- official API upload where the platform supports it cleanly
- Chromium profile automation for browser-based publishing

## 1. High-Level Concept
- `1 Video -> N Platforms -> M Accounts`
- API when possible, Chromium profile when needed
- Browser-first MVP: a platform account is represented by one Chromium profile
- Asynchronous publishing: videos are generated first, then a specialized worker handles distribution to requested destinations

---

## 2. Database Schema

### `connected_accounts`
Stores the account connection state for each platform account.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID/TEXT (PK) | Unique account identifier, for example `yt-main`. |
| `platform` | TEXT | `youtube`, `tiktok`, `instagram`. |
| `display_name` | TEXT | Human readable name, for example "My Gaming Channel". |
| `auth_method` | TEXT | `api` or `chromium_profile`. |
| `access_token` | TEXT (Encrypted) | Current valid OAuth access token, if the platform uses API auth. |
| `refresh_token` | TEXT (Encrypted) | Token used to get new access tokens, if available. |
| `profile_path` | TEXT | Chromium profile directory used for browser automation. |
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
| `external_id` | TEXT | The ID of the post on the platform, for example YouTube Video ID. |
| `error` | TEXT | Error message if failed. |

---

## 3. Account Linking Flow

### API path
1. User clicks "Connect Account" for a platform that supports API upload.
2. Backend redirects to the platform authorization URL.
3. Platform redirects back with `code`.
4. Backend exchanges `code` for tokens.
5. Backend stores encrypted token state.

### Chromium profile path
1. User clicks "Connect Account" for a platform that will be driven by browser automation.
2. Backend creates or reuses a Chromium profile directory for the pair `(platform, email)`.
3. User logs in once inside that profile.
4. Backend saves the profile path and marks the account as connected.
5. Future uploads reuse the same profile.

For MVP, the Chromium profile path is the default browser integration, so the UI can treat `chromium_profile` as the main non-API option.

---

## 4. User Interface

### Account Management
- A `Settings` or `Integrations` page listing all connected accounts.
- Status indicator for connected or re-auth needed.
- Buttons to `Add Account` or `Disconnect`.

### Job Creation Form
A dynamic checkbox list populated from `connected_accounts`:

```text
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

We follow the adapter pattern to handle different platform APIs through a unified interface.

### New Port: `Publisher`
```go
type Publisher interface {
    Publish(ctx context.Context, videoPath string, metadata Metadata, account ConnectedAccount) (externalID string, err error)
}
```

### Execution Flow
1. `GeneratorService` finishes video assembly and local storage.
2. It creates entries in `distribution_jobs` for every checked account.
3. A publisher worker picks up pending distribution jobs.
4. It fetches encrypted tokens or Chromium profile paths, then calls the corresponding platform adapter.
5. If one platform fails, it updates the `distribution_jobs` error field but does not affect other platforms.

---

## 6. Implementation Payload Example

Request to `/api/generate`:

```json
{
  "niche": "stoicism",
  "topic": "how to control your mind",
  "voice_type": "en-US-Standard-A",
  "destinations": [
    { "platform": "youtube", "account_id": "yt-main" },
    { "platform": "tiktok", "account_id": "tt-brand" }
  ]
}
```
