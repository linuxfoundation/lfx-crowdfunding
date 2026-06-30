# Frontend Developer Guide — Initiative Announcements API

> **Audience:** Frontend / BFF developers building the LFX Crowdfunding UI.  
> **Covers:** Listing, creating, updating, and deleting announcements on an initiative, including the Nuxt BFF proxy pattern and full payload reference.

---

## Table of Contents

1. [Quick-start cheat sheet](#1-quick-start-cheat-sheet)
2. [Authentication](#2-authentication)
3. [Data model](#3-data-model)
4. [List announcements — GET /v1/initiatives/{id}/announcements](#4-list-announcements)
5. [Create an announcement — POST /v1/me/initiatives/{id}/announcements](#5-create-an-announcement)
6. [Update an announcement — PUT /v1/me/initiatives/{id}/announcements/{announcementId}](#6-update-an-announcement)
7. [Delete an announcement — DELETE /v1/me/initiatives/{id}/announcements/{announcementId}](#7-delete-an-announcement)
8. [Pagination](#8-pagination)
9. [Validation rules](#9-validation-rules)
10. [Error reference](#10-error-reference)
11. [Nuxt BFF proxy pattern](#11-nuxt-bff-proxy-pattern)
12. [End-to-end worked example](#12-end-to-end-worked-example)

---

## 1. Quick-start cheat sheet

| Action | Method | Path | Auth? | Who can call |
|--------|--------|------|-------|--------------|
| List announcements | `GET` | `/v1/initiatives/{id}/announcements` | No | Anyone (public) |
| **Create announcement** | `POST` | `/v1/me/initiatives/{id}/announcements` | **Yes** | Initiative owner |
| **Update announcement** | `PUT` | `/v1/me/initiatives/{id}/announcements/{announcementId}` | **Yes** | Initiative owner |
| **Delete announcement** | `DELETE` | `/v1/me/initiatives/{id}/announcements/{announcementId}` | **Yes** | Initiative owner |

`{id}` — the initiative UUID  
`{announcementId}` — the announcement UUID  
All write endpoints live under `/v1/me/` — same auth group as initiative mutations.

---

## 2. Authentication

**List** is public — no token required.

**Create / Update / Delete** require a valid Auth0 JWT as a `Bearer` token:

```http
Authorization: Bearer <id_token>
```

The caller must be the **owner** of the initiative. If the token is valid but belongs to a different user, the API returns `403 Forbidden`. The BFF should forward the session token it holds via PKCE — never expose the token to the browser directly.

---

## 3. Data model

Every announcement has the following shape:

```typescript
interface Announcement {
  // Server-set — do not send in request bodies
  id: string            // UUID, assigned by the server
  initiative_id: string // UUID — taken from the URL path
  created_by: string    // set to the authenticated user's username
  created_on: string    // ISO 8601 timestamp, set on insert
  updated_on: string    // ISO 8601 timestamp, updated on every PUT

  // Caller-supplied
  title: string         // plain text, max 255 chars
  description: string   // HTML allowed
}
```

> `id`, `initiative_id`, `created_by`, `created_on`, and `updated_on` are all set automatically by the backend.
> Request bodies for Create and Update only need `title` and `description`.

---

## 4. List announcements

```
GET /v1/initiatives/{id}/announcements
```

**Auth:** None  
**Query params:** `limit` (default 20, max 100), `offset` (default 0)

### Response — 200 OK

```json
{
  "data": [
    {
      "id": "a1b2c3d4-0000-0000-0000-000000000001",
      "initiative_id": "f44da17d-70da-45f3-8f9b-0ae733494167",
      "created_by": "jane.smith",
      "title": "Q2 funding milestone reached",
      "description": "<p>We hit <strong>50% of our goal</strong>! Thank you to all backers.</p>",
      "created_on": "2026-06-30T10:00:00Z",
      "updated_on": "2026-06-30T10:00:00Z"
    }
  ],
  "meta": {
    "total": 1,
    "limit": 20,
    "offset": 0
  }
}
```

Results are ordered **newest first** (`created_on` descending).

### Error cases

| Status | Cause |
|--------|-------|
| `404 Not Found` | The `{id}` does not match any initiative |
| `400 Bad Request` | `limit` or `offset` is not a valid integer |

---

## 5. Create an announcement

```
POST /v1/me/initiatives/{id}/announcements
```

**Auth:** JWT required — caller must own the initiative  
**Content-Type:** `application/json`

### Request body

```json
{
  "title": "Q2 funding milestone reached",
  "description": "<p>We hit <strong>50% of our goal</strong>! Thank you to all backers.</p>"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `title` | string | Yes | Non-empty, ≤ 255 characters |
| `description` | string | Yes | Non-empty, no maximum length |

### Response — 201 Created

```json
{
  "id": "a1b2c3d4-0000-0000-0000-000000000001",
  "initiative_id": "f44da17d-70da-45f3-8f9b-0ae733494167",
  "created_by": "jane.smith",
  "title": "Q2 funding milestone reached",
  "description": "<p>We hit <strong>50% of our goal</strong>! Thank you to all backers.</p>",
  "created_on": "2026-06-30T10:00:00Z",
  "updated_on": "2026-06-30T10:00:00Z"
}
```

### Error cases

| Status | Cause |
|--------|-------|
| `400 Bad Request` | `title` or `description` is missing or empty; `title` exceeds 255 chars; invalid JSON |
| `401 Unauthorized` | No JWT provided |
| `403 Forbidden` | JWT valid but caller is not the initiative owner |
| `404 Not Found` | `{id}` does not match any initiative |

---

## 6. Update an announcement

```
PUT /v1/me/initiatives/{id}/announcements/{announcementId}
```

**Auth:** JWT required — caller must own the initiative  
**Content-Type:** `application/json`

> **Full replacement** — both `title` and `description` must be sent even if only one changed.

### Request body

```json
{
  "title": "Q2 funding milestone reached (updated)",
  "description": "<p>Correction: we actually hit <strong>55%</strong>. Thank you!</p>"
}
```

Constraints are identical to [Create](#5-create-an-announcement).

### Response — 200 OK

Returns the updated announcement in the same shape as Create.

### Error cases

| Status | Cause |
|--------|-------|
| `400 Bad Request` | Validation failure (same rules as Create) |
| `401 Unauthorized` | No JWT provided |
| `403 Forbidden` | Caller does not own the initiative |
| `404 Not Found` | `{id}` (initiative) or `{announcementId}` does not exist, or the announcement does not belong to this initiative |

---

## 7. Delete an announcement

```
DELETE /v1/me/initiatives/{id}/announcements/{announcementId}
```

**Auth:** JWT required — caller must own the initiative  
**Request body:** None

### Response — 204 No Content

Empty body on success.

### Error cases

| Status | Cause |
|--------|-------|
| `401 Unauthorized` | No JWT provided |
| `403 Forbidden` | Caller does not own the initiative |
| `404 Not Found` | Announcement not found or does not belong to this initiative |

---

## 8. Pagination

The List endpoint supports standard cursor-free pagination:

```
GET /v1/initiatives/{id}/announcements?limit=10&offset=0   # page 1
GET /v1/initiatives/{id}/announcements?limit=10&offset=10  # page 2
GET /v1/initiatives/{id}/announcements?limit=10&offset=20  # page 3
```

`meta.total` always contains the full count so you can compute `hasNextPage`:

```typescript
const hasNextPage = meta.offset + meta.limit < meta.total
```

Limits are server-clamped: `limit > 100` is silently reduced to 20 (the default). Pass an explicit limit if you need more than 20 per page.

---

## 9. Validation rules

| Field | Rule | Error message |
|-------|------|---------------|
| `title` | Required | `title is required` |
| `title` | ≤ 255 characters | `title must be 255 characters or fewer` |
| `description` | Required | `description is required` |

The API returns `400 Bad Request` with the message in the `error` field:

```json
{ "error": "invalid input: title must be 255 characters or fewer" }
```

---

## 10. Error reference

All errors follow the same envelope:

```json
{ "error": "<human-readable message>" }
```

| HTTP status | `error` value | Meaning |
|-------------|--------------|---------|
| `400` | Validation message | Request body invalid |
| `401` | `"unauthorized"` | Missing or invalid JWT |
| `403` | `"forbidden"` | Valid JWT but caller is not the owner |
| `404` | `"not found"` | Initiative or announcement does not exist |
| `500` | `"internal server error"` | Unexpected server error |

---

## 11. Nuxt BFF proxy pattern

The Nuxt server acts as a BFF — it forwards requests to the Go API and attaches the session token. Below are minimal server route examples.

### `server/api/initiatives/[id]/announcements/index.ts`

```typescript
// GET  /api/initiatives/:id/announcements  → proxies to Go public endpoint
// POST /api/initiatives/:id/announcements  → proxies to Go /v1/me endpoint (auth required)
export default defineEventHandler(async (event) => {
  const initiativeId = getRouterParam(event, 'id')
  const method = getMethod(event)

  if (method === 'GET') {
    const query = getQuery(event)
    return proxyRequest(event, `/v1/initiatives/${initiativeId}/announcements`, {
      query,
    })
  }

  if (method === 'POST') {
    const session = await requireUserSession(event)
    const body = await readBody(event)
    return proxyRequest(event, `/v1/me/initiatives/${initiativeId}/announcements`, {
      method: 'POST',
      body,
      headers: { Authorization: `Bearer ${session.token}` },
    })
  }
})
```

### `server/api/initiatives/[id]/announcements/[announcementId].ts`

```typescript
// PUT    /api/initiatives/:id/announcements/:announcementId
// DELETE /api/initiatives/:id/announcements/:announcementId
export default defineEventHandler(async (event) => {
  const initiativeId = getRouterParam(event, 'id')
  const announcementId = getRouterParam(event, 'announcementId')
  const method = getMethod(event)
  const session = await requireUserSession(event)

  const upstreamPath = `/v1/me/initiatives/${initiativeId}/announcements/${announcementId}`

  if (method === 'PUT') {
    const body = await readBody(event)
    return proxyRequest(event, upstreamPath, {
      method: 'PUT',
      body,
      headers: { Authorization: `Bearer ${session.token}` },
    })
  }

  if (method === 'DELETE') {
    return proxyRequest(event, upstreamPath, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${session.token}` },
    })
  }
})
```

> Replace `proxyRequest` with whatever fetch/proxy helper your server routes use. The key point is that the session token is attached **server-side only** — it never appears in a browser network request.

---

## 12. End-to-end worked example

This walks through the full lifecycle for an initiative owner posting their first announcement.

### Step 1 — Load the list on the initiative detail page

```typescript
// composables/useAnnouncements.ts
const { data } = await useFetch(`/api/initiatives/${initiativeId}/announcements`, {
  query: { limit: 10, offset: 0 },
})
// data.value = { data: Announcement[], meta: { total, limit, offset } }
```

### Step 2 — Owner creates an announcement

```typescript
const { data: created, error } = await useFetch(
  `/api/initiatives/${initiativeId}/announcements`,
  {
    method: 'POST',
    body: {
      title: 'Q2 milestone reached!',
      description: '<p>We hit <strong>50%</strong> of our funding goal.</p>',
    },
  }
)

if (error.value?.statusCode === 400) {
  // show validation message from error.value.data.error
}
```

### Step 3 — Owner edits the announcement

```typescript
await $fetch(`/api/initiatives/${initiativeId}/announcements/${announcementId}`, {
  method: 'PUT',
  body: {
    title: 'Q2 milestone reached (corrected)',
    description: '<p>We actually hit <strong>55%</strong>. Thank you!</p>',
  },
})
```

### Step 4 — Owner deletes the announcement

```typescript
await $fetch(`/api/initiatives/${initiativeId}/announcements/${announcementId}`, {
  method: 'DELETE',
})
// 204 No Content — no response body
```

### Rendering HTML safely

`description` may contain HTML. Always sanitise before rendering:

```vue
<template>
  <div v-html="sanitised" />
</template>

<script setup lang="ts">
import DOMPurify from 'dompurify'
const props = defineProps<{ description: string }>()
const sanitised = computed(() => DOMPurify.sanitize(props.description))
</script>
```

Install DOMPurify: `pnpm add dompurify @types/dompurify`
