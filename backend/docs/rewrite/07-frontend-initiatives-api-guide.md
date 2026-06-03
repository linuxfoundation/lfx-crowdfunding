# Frontend Developer Guide — Initiatives API

> **Audience:** Frontend / BFF developers building the LFX Crowdfunding UI.  
> **Covers:** Creating and updating initiatives, the full payload for every initiative type, and end-to-end handling of the email-based approval flow.

---

## Table of Contents

1. [Quick-start cheat sheet](#1-quick-start-cheat-sheet)
2. [Authentication](#2-authentication)
3. [Initiative types and statuses](#3-initiative-types-and-statuses)
4. [Creating an initiative — POST /v1/initiatives](#4-creating-an-initiative)
5. [Full payload reference by type](#5-full-payload-reference-by-type)
   - 5.1 project
   - 5.2 event
   - 5.3 mentorship
   - 5.4 security_audit
   - 5.5 general_fund / other
   - 5.6 ostif
6. [Updating an initiative — PATCH /v1/initiatives/{id}](#6-updating-an-initiative)
7. [Child-table replace semantics (IMPORTANT)](#7-child-table-replace-semantics)
8. [Status lifecycle and the approval workflow](#8-status-lifecycle-and-the-approval-workflow)
9. [The email approval flow — step by step](#9-the-email-approval-flow)
10. [Implementing the approve or decline page](#10-implementing-the-approve-or-decline-page)
11. [Error reference](#11-error-reference)
12. [End-to-end worked example](#12-end-to-end-worked-example)

---

## 1. Quick-start cheat sheet

| Action | Method | Path | Auth? |
|--------|--------|------|-------|
| List initiatives (public) | `GET` | `/v1/initiatives` | No |
| Get one initiative (public) | `GET` | `/v1/initiatives/{slug-or-uuid}` | No |
| **Create initiative** | `POST` | `/v1/initiatives` | **Yes** (user JWT) |
| **Update initiative** | `PATCH` | `/v1/initiatives/{uuid}` | **Yes** (owner JWT) |
| Delete initiative | `DELETE` | `/v1/initiatives/{uuid}` | **Yes** (owner JWT) |
| **Approve / decline** | `POST` | `/v1/initiatives/{uuid}/process-approval/{action}` | **Yes** (approver JWT) |

Base URL (production): `https://api-gw.platform.linuxfoundation.org`  
All endpoints are under `/v1/initiatives`.

---

## 2. Authentication

All write endpoints require a valid **Auth0 JWT** passed as a `Bearer` token:

```http
Authorization: Bearer <id_token>
```

The BFF (Nuxt server) should forward the session token it obtained via PKCE. If the token is missing or expired the API returns `401 Unauthorized`.

The **approval** endpoint additionally requires that the authenticated user's `username` appears in the server-side `ALLOWED_APPROVERS` list — if not, it returns `403 Forbidden`. Regular users can never approve their own initiatives.

---

## 3. Initiative types and statuses

### 3.1 Valid `initiative_type` values

| Value | Description |
|-------|-------------|
| `project` | Open-source software project |
| `event` | Conference, summit, or meetup |
| `mentorship` | Mentorship programme |
| `security_audit` | Security / OSTIF audit |
| `general_fund` | General fundraising fund |
| `ostif` | OSTIF-specific initiative |
| `other` | Anything that doesn't fit elsewhere |

> `community` and `travel_fund` existed in the legacy system but are **not accepted** by this API.

### 3.2 Valid `status` values

| Status | Who sets it | Meaning |
|--------|------------|---------|
| `submitted` | **Auto-set by API on create** | Awaiting reviewer action |
| `pending` | Approval workflow only | Under active review |
| `published` | Approval workflow only | Live and visible to the public |
| `declined` | Approval workflow only | Rejected by a reviewer |
| `hidden` | Owner (via PATCH) | Temporarily hidden from public listing |

> **You cannot set `published`, `declined`, or `pending` via the PATCH endpoint.** Attempting to do so returns `403 Forbidden`. These statuses are exclusively set by the approval workflow (`/process-approval/{action}`).

### 3.3 Status lifecycle

```
                  ┌────────────────────────────────────────┐
                  │                                        │
  User submits    │                                        ▼
  POST /v1/initiatives ──► submitted ──► pending ──► published
                                    │
                                    └──────────────► declined
                                          ▲
                                          │
                          Reviewer POSTs /process-approval/approve
                              or /process-approval/decline
```

**In plain English:**

1. User fills in the form and hits Submit → `POST /v1/initiatives` → status is auto-set to `submitted`.
2. An email is sent to the reviewer inbox with a link to approve or decline.
3. Reviewer clicks a link in the email → a page in your frontend authenticates them and calls `POST /v1/initiatives/{id}/process-approval/approve` (or `/decline`).
4. If approved → status becomes `published` and the initiative is live.
5. If declined → status becomes `declined` and an email is sent to the owner.

---

## 4. Creating an initiative

### Endpoint

```
POST /v1/initiatives
Content-Type: application/json
Authorization: Bearer <token>
```

### Response

- **201 Created** — returns the full `Initiative` object.
- **400 Bad Request** — validation error (e.g. missing `name`, unknown `initiative_type`).
- **401 Unauthorized** — no / invalid token.

### What happens on a successful create

1. A UUID is generated for the initiative.
2. A Stripe Product is created (for donation processing).
3. The initiative is saved to the database with status `submitted`.
4. An email is sent to the reviewer inbox (configured via `MANDRILL_NOTIFICATION_EMAIL`) with approve/decline deep-links.
5. The full initiative object is returned.

> **Note:** Email delivery failure is **non-fatal**. The initiative is still created and `201` is still returned even if the email could not be sent.

### Minimum required payload

Every initiative type requires at a minimum:

```json
{
  "initiative_type": "project",
  "name": "My Project",
  "accept_funding": true
}
```

`slug` is optional — if omitted the UI can let the owner set it later via PATCH.

---

## 5. Full payload reference by type

The top-level fields are shared by all types. Child-table fields are type-specific.

### 5.1 `project`

Projects can have goals, beneficiaries, custom websites, and contributors.

```json
{
  "initiative_type": "project",
  "name": "Kubernetes Dashboard",
  "slug": "kubernetes-dashboard",
  "description": "A web-based UI for Kubernetes clusters.",
  "industry": "Cloud Native",
  "color": "#326CE5",
  "logo_url": "https://cdn.example.com/logo.png",
  "website_url": "https://kubernetes.io",
  "coc_url": "https://kubernetes.io/code-of-conduct",
  "accept_funding": true,

  "goals": [
    {
      "name": "Core Development",
      "amount_in_cents": 5000000,
      "allocation": "engineering",
      "description": "Fund core maintainers",
      "repo_link": "https://github.com/kubernetes/dashboard",
      "color": "#326CE5",
      "icon": "code",
      "sort_order": 0
    },
    {
      "name": "Documentation",
      "amount_in_cents": 1000000,
      "sort_order": 1
    }
  ],

  "beneficiaries": [
    { "name": "Alice Smith", "email": "alice@example.com" },
    { "name": "Bob Jones",  "email": "bob@example.com" }
  ],

  "custom_websites": [
    { "name": "Docs",  "url": "https://docs.example.com" },
    { "name": "Forum", "url": "https://discuss.example.com" }
  ],

  "contributors": [
    { "name": "Carol White", "email": "carol@example.com" },
    { "name": "Dan Brown",   "email": "dan@example.com" }
  ]
}
```

---

### 5.2 `event`

Events add venue / date fields and sponsorship tiers. The `entity_details` map holds any extra key-value metadata specific to your event.

```json
{
  "initiative_type": "event",
  "name": "KubeCon + CloudNativeCon 2026",
  "slug": "kubecon-2026",
  "description": "The premier conference for Kubernetes and cloud-native technologies.",
  "industry": "Cloud Native",
  "color": "#FF6B6B",
  "logo_url": "https://cdn.example.com/kubecon.png",
  "website_url": "https://events.linuxfoundation.org/kubecon",
  "accept_funding": true,

  "eventbrite_url":   "https://eventbrite.com/e/kubecon-2026-123456",
  "application_url":  "https://cfp.kubecon.io",
  "event_start_date": "2026-11-10T00:00:00Z",
  "event_end_date":   "2026-11-13T00:00:00Z",
  "country": "US",
  "city":    "Chicago",
  "is_online": false,

  "goals": [
    {
      "name": "Scholarship Fund",
      "amount_in_cents": 25000000,
      "description": "Fund diversity scholarships",
      "sort_order": 0
    }
  ],

  "sponsorship_tiers": [
    {
      "name": "Platinum",
      "description": "Top-tier sponsorship with keynote slot",
      "minimum": 10000000,
      "color": "#E5E4E2",
      "icon": "star",
      "sort_order": 0
    },
    {
      "name": "Gold",
      "minimum": 5000000,
      "sort_order": 1
    },
    {
      "name": "Silver",
      "minimum": 2500000,
      "sort_order": 2
    }
  ],

  "entity_details": {
    "venue":          "McCormick Place",
    "expected_attendees": "12000",
    "cncf_project":   "true"
  }
}
```

---

### 5.3 `mentorship`

Mentorship programmes have mentors and a `program_info` block with terms, skills, and optionally a custom term window.

```json
{
  "initiative_type": "mentorship",
  "name": "LFX Mentorship — Go Networking",
  "slug": "lfx-mentorship-go-networking",
  "description": "12-week mentorship covering gRPC, HTTP/3, and network programming in Go.",
  "industry": "Open Source",
  "accept_funding": true,

  "mentors": [
    {
      "name":         "Dr. Jane Roe",
      "email":        "jane@example.com",
      "avatar_url":   "https://cdn.example.com/jane.jpg",
      "introduction": "10 years of systems programming, core contributor to gRPC-Go."
    }
  ],

  "program_info": {
    "terms": ["Spring 2026", "Fall 2026"],
    "skills": ["Go", "gRPC", "Networking", "Kubernetes"],
    "terms_conditions": true,

    "custom_term": {
      "term_name":   "Summer 2026",
      "start_month": "June",
      "end_month":   "August",
      "year":        2026
    }
  }
}
```

> `custom_term` is optional. Omit the `custom_term` key entirely if you only use standard terms.  
> `terms_conditions` is a boolean flag indicating the submitter accepted the programme T&Cs.

---

### 5.4 `security_audit`

Security audits use the same fields as a `project` — there are no type-specific child tables. Use goals to describe funding targets.

```json
{
  "initiative_type": "security_audit",
  "name": "OpenSSL Security Audit 2026",
  "slug": "openssl-security-audit-2026",
  "description": "Comprehensive security audit of OpenSSL 3.x.",
  "industry": "Security",
  "accept_funding": true,

  "goals": [
    {
      "name": "Audit Phase 1 — Code Review",
      "amount_in_cents": 8000000,
      "sort_order": 0
    },
    {
      "name": "Audit Phase 2 — Penetration Testing",
      "amount_in_cents": 4000000,
      "sort_order": 1
    }
  ]
}
```

---

### 5.5 `general_fund` / `other`

These types support the full set of shared fields plus goals, beneficiaries, and custom websites. There are no type-specific child tables.

```json
{
  "initiative_type": "general_fund",
  "name": "CNCF General Fund",
  "slug": "cncf-general-fund",
  "description": "Supports CNCF operational costs and community programmes.",
  "accept_funding": true,

  "goals": [
    {
      "name": "Operations",
      "amount_in_cents": 50000000,
      "sort_order": 0
    }
  ]
}
```

---

### 5.6 `ostif`

OSTIF initiatives have an `ostif_detail` block and a `contacts` array. These replace `contributors`/`mentors` for this type.

```json
{
  "initiative_type": "ostif",
  "name": "curl Security Audit",
  "slug": "curl-security-audit",
  "description": "OSTIF-coordinated security audit of the curl library.",
  "accept_funding": true,

  "ostif_detail": {
    "monetization_strategy":     "Donations from companies that depend on curl",
    "current_security_strategy": "Manual code review; no automated fuzzing",
    "license_type":              "MIT",
    "total_budget_in_cents":     1200000,
    "terms_conditions":          true
  },

  "contacts": [
    {
      "contact_type":              "primary",
      "first_name":                "Jane",
      "last_name":                 "Doe",
      "email":                     "jane@example.com",
      "phone_number":              "+1-555-123-4567",
      "preferred_contact_method":  "email"
    },
    {
      "contact_type": "technical_lead",
      "first_name":   "John",
      "email":        "john@example.com"
    }
  ]
}
```

**Valid `contact_type` values:** `primary`, `secondary`, `technical_lead`

---

## 6. Updating an initiative

### Endpoint

```
PATCH /v1/initiatives/{uuid}
Content-Type: application/json
Authorization: Bearer <token>
```

> The `{uuid}` must be the initiative's UUID, not the slug.

### Response

- **200 OK** — returns the full updated `Initiative` object.
- **400 Bad Request** — validation error.
- **403 Forbidden** — caller is not the initiative owner, or tried to set a restricted status.
- **404 Not Found** — initiative does not exist.

### Pointer semantics — only send what you want to change

The PATCH endpoint uses **pointer semantics**:

| Field type | Omitted / `null` in JSON | Explicit value |
|------------|--------------------------|----------------|
| Scalar field (e.g. `"name"`) | **Unchanged** | **Replaced** with new value |
| Child table array (e.g. `"goals"`) | **Unchanged** (no DB writes) | **Replaces all rows** (even if empty array `[]`) |
| Child table pointer (e.g. `"program_info"`) | **Unchanged** | **Replaces all sub-tables** |

**Example: rename an initiative, change nothing else**

```json
{
  "name": "My Renamed Project"
}
```

**Example: update description and replace all goals**

```json
{
  "description": "Updated description after community vote.",
  "goals": [
    { "name": "New Goal A", "amount_in_cents": 3000000, "sort_order": 0 },
    { "name": "New Goal B", "amount_in_cents": 1500000, "sort_order": 1 }
  ]
}
```

**Example: delete all goals (send an empty array)**

```json
{
  "goals": []
}
```

> ⚠️ Sending `"goals": []` **deletes all existing goals**. Omitting the `goals` key entirely leaves existing goals untouched.

### Statuses you CAN set via PATCH

Only `hidden` and `submitted` can be set by the owner via PATCH:

```json
{ "status": "hidden" }
```

Attempting to set `published`, `declined`, or `pending` returns `403 Forbidden`.

---

## 7. Child-table replace semantics

This is the **most important concept** to understand for the update flow.

```
PATCH sends "goals": null / omitted
    ──► Database: goals table is NOT touched
    ──► UI: safe to call PATCH with only the fields you actually edited

PATCH sends "goals": [...]  (one or more items)
    ──► Database: DELETE all existing goals, INSERT new ones
    ──► UI: must always send the COMPLETE desired list

PATCH sends "goals": []  (empty array)
    ──► Database: DELETE all existing goals, INSERT nothing
    ──► UI: this is how you remove all goals
```

The same rule applies to every child table:
`beneficiaries`, `custom_websites`, `contributors`, `mentors`, `contacts`, `sponsorship_tiers`, `program_info`, `ostif_detail`, `entity_details`.

### Practical UI guidance

If your edit form allows adding/removing goals:

```javascript
// Build the PATCH body carefully:
const patch = {};

if (nameChanged)  patch.name  = newName;
if (colorChanged) patch.color = newColor;

// Only include goals if the user actually changed them in the UI:
if (goalsChanged) {
  // Send the COMPLETE current list including unchanged goals —
  // not just the delta.
  patch.goals = allGoals.map(g => ({
    name:           g.name,
    amount_in_cents: g.amountInCents,
    sort_order:     g.sortOrder,
    // ... all other fields
  }));
}

await $fetch(`/v1/initiatives/${id}`, { method: 'PATCH', body: patch });
```

---

## 8. Status lifecycle and the approval workflow

### What the API automatically does

When an initiative is created (`POST /v1/initiatives`):

1. Status is **always** set to `submitted` by the API — you cannot override this.
2. The API fetches the owner's profile (name + email) from the database.
3. A **"Submitted for Review" email** is sent to the reviewer inbox (`MANDRILL_NOTIFICATION_EMAIL`) containing:
   - The submitter's name and email
   - The initiative name
   - A **View** link: `https://crowdfunding.linuxfoundation.org/initiatives/{slug}`
   - An **Approve** link: `https://crowdfunding.linuxfoundation.org/initiatives/{slug}/process-approval/approve`
   - A **Decline** link: `https://crowdfunding.linuxfoundation.org/initiatives/{slug}/process-approval/decline`

> Email failure is **non-fatal** — the API still returns `201` even if Mandrill is unreachable or not configured.

---

## 9. The email approval flow

This is the complete sequence from submission to a live initiative:

```
                                                      Mandrill
  Submitter                Backend API              Email Service           Reviewer Inbox
     │                         │                         │                       │
     │  POST /v1/initiatives   │                         │                       │
     │────────────────────────►│                         │                       │
     │                         │ save to DB (submitted)  │                       │
     │                         │─────────────────────────►                       │
     │                         │                         │ "Submitted for Review"│
     │                         │                         │ email with links ─────►
     │  201 { ...initiative }  │                         │                       │
     │◄────────────────────────│                         │                       │
     │                         │                         │    Reviewer clicks    │
     │                         │                         │    "Approve" link     │
     │                         │                         │◄──────────────────────│
     │                         │                         │
     │                     Browser opens the Approve page
     │                     (your frontend route)
     │                         │
     │                         │ POST /v1/initiatives/{id}/process-approval/approve
     │                         │ (with reviewer's JWT)
     │                         │
     │                         │ status → published
     │                         │
     │                         │ "Approved" email sent to submitter
     │                         │
     │  Submitter receives      │
     │  "Your initiative is live" email
```

### Email templates used

| Event | Mandrill template slug | Recipient |
|-------|----------------------|-----------|
| New submission | `communitybridge-review-mentorship-submission` | Reviewer inbox |
| Approved | `admin-mentorship-submission-approved` | Initiative owner |
| Declined | `admin-mentorship-submission-rejected` | Initiative owner |

---

## 10. Implementing the approve or decline page

When a reviewer clicks a link in the email they are directed to a URL like:

```
https://crowdfunding.linuxfoundation.org/initiatives/{slug}/process-approval/approve
```

Your frontend needs to handle this route. Here is exactly what the page must do:

### Step 1 — Check that the user is logged in

The reviewer must be authenticated. If they are not logged in, redirect them to the Auth0 login page, then return them to this URL after login.

```
if (!user) {
  redirectToLogin({ returnTo: currentRoute })
}
```

### Step 2 — Extract the slug and action from the URL

```javascript
// In a Nuxt page component at:
// pages/initiatives/[slug]/process-approval/[action].vue

const route = useRoute()
const slug   = route.params.slug    // e.g. "kubernetes-dashboard"
const action = route.params.action  // "approve" or "decline"
```

### Step 3 — Resolve the slug to a UUID

The approval API endpoint takes a **UUID**, not a slug. You need to fetch the initiative first to get its ID.

```javascript
const initiative = await $fetch(`/v1/initiatives/${slug}`)
const initiativeId = initiative.id
```

> If this returns `404`, the initiative does not exist or is not yet published. Show an error page.

### Step 4 — Call the approval endpoint

```javascript
// Action must be "approve" or "decline"
const result = await $fetch(
  `/v1/initiatives/${initiativeId}/process-approval/${action}`,
  {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  }
)
```

### Step 5 — Handle the response

| HTTP status | Meaning | What to show |
|------------|---------|-------------|
| `200 OK` | Success — initiative is now `published` or `declined` | "Initiative approved / declined" confirmation |
| `400 Bad Request` | Action was not `approve` or `decline` | "Invalid action" error page |
| `401 Unauthorized` | Token missing or expired | Redirect to login |
| `403 Forbidden` | Reviewer is not in `ALLOWED_APPROVERS` | "You do not have permission to approve initiatives" |
| `404 Not Found` | Initiative not found or already in a terminal state | "Initiative not found" |
| `422` / `400` with `"invalid input"` | Initiative is in a state that cannot be approved (e.g. already `published`) | "This initiative has already been reviewed" |

### Complete Nuxt page example

```vue
<!-- pages/initiatives/[slug]/process-approval/[action].vue -->
<script setup lang="ts">
const route   = useRoute()
const slug    = route.params.slug as string
const action  = route.params.action as string   // "approve" | "decline"

// Redirect to login if not authenticated (BFF handles this via middleware)
definePageMeta({ middleware: 'auth' })

const { data: initiative, error: fetchError } = await useFetch(
  `/api/initiatives/${slug}`    // Nuxt server route proxies to the Go API
)

if (fetchError.value || !initiative.value) {
  throw createError({ statusCode: 404, message: 'Initiative not found' })
}

const result    = ref<Record<string, unknown> | null>(null)
const apiError  = ref<string | null>(null)
const loading   = ref(false)

async function confirm() {
  loading.value = true
  try {
    result.value = await $fetch(
      `/api/initiatives/${initiative.value!.id}/process-approval/${action}`,
      { method: 'POST' }   // BFF forwards JWT from session cookie
    )
  } catch (err: any) {
    apiError.value = err?.data?.error ?? 'An unexpected error occurred'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <template v-if="result">
      <h1>
        Initiative {{ action === 'approve' ? 'approved' : 'declined' }}
      </h1>
      <p>
        {{ initiative!.name }} has been {{ action === 'approve' ? 'published' : 'declined' }}.
      </p>
    </template>

    <template v-else-if="apiError">
      <h1>Something went wrong</h1>
      <p>{{ apiError }}</p>
    </template>

    <template v-else>
      <h1>{{ action === 'approve' ? 'Approve' : 'Decline' }} initiative</h1>
      <p>
        You are about to
        <strong>{{ action === 'approve' ? 'approve' : 'decline' }}</strong>
        "<strong>{{ initiative!.name }}</strong>".
        This action will send an email notification to the owner.
      </p>
      <button :disabled="loading" @click="confirm">
        {{ loading ? 'Processing…' : 'Confirm' }}
      </button>
    </template>
  </div>
</template>
```

### Step 6 — Nuxt BFF server route (proxy)

The approval endpoint requires a JWT. The BFF server route extracts the session token and forwards it:

```typescript
// server/api/initiatives/[id]/process-approval/[action].post.ts
import { defineEventHandler, getRouterParam, createError } from 'h3'

export default defineEventHandler(async (event) => {
  const session     = await getUserSession(event)
  const accessToken = session?.token?.access_token

  if (!accessToken) {
    throw createError({ statusCode: 401, message: 'Not authenticated' })
  }

  const id     = getRouterParam(event, 'id')!
  const action = getRouterParam(event, 'action')!
  const apiBase = useRuntimeConfig().apiBase   // e.g. http://initiatives-api:8080

  const response = await $fetch(
    `${apiBase}/v1/initiatives/${id}/process-approval/${action}`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
    }
  )

  return response
})
```

---

## 11. Error reference

All errors are returned as:

```json
{
  "error": "human-readable message"
}
```

| HTTP | When you see it | Fix |
|------|----------------|-----|
| `400` | Missing required field, invalid `initiative_type`, child-table row missing required field (e.g. `goal.name` is empty, `contact.contact_type` is empty, `custom_website.url` is empty) | Check the `error` field for the exact field name |
| `401` | JWT missing, expired, or invalid | Re-authenticate; refresh the session token |
| `403` | Updating another user's initiative; trying to set a restricted status (`published`, `declined`, `pending`) directly; approver not in `ALLOWED_APPROVERS` | Use the approval flow; check approver permissions |
| `404` | Initiative not found; GET of a non-published initiative | Verify the ID/slug is correct |
| `429` | Rate limited | Back off and retry with exponential backoff |
| `503` | Upstream service (Ledger/Stripe) is unavailable | Display "service temporarily unavailable" |

### Validation rules for child-table required fields

| Field | Requirement |
|-------|------------|
| `goals[].name` | Required — must be non-empty |
| `custom_websites[].url` | Required — must be non-empty |
| `contacts[].contact_type` | Required — must be `primary`, `secondary`, or `technical_lead` |

---

## 12. End-to-end worked example

This example walks through creating a `project` initiative and then simulating the approval flow.

### Step 1 — Create the initiative (from the UI)

```http
POST /v1/initiatives
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6...
Content-Type: application/json

{
  "initiative_type": "project",
  "name": "My Awesome Project",
  "slug": "my-awesome-project",
  "description": "This project does amazing things.",
  "accept_funding": true,
  "goals": [
    { "name": "Core Development", "amount_in_cents": 10000000, "sort_order": 0 }
  ]
}
```

**Response `201 Created`:**

```json
{
  "id":               "b4a1e2c3-dead-beef-1234-56789abcdef0",
  "initiative_type":  "project",
  "name":             "My Awesome Project",
  "slug":             "my-awesome-project",
  "status":           "submitted",
  "accept_funding":   true,
  "goals": [
    {
      "id":             "a1b2c3d4-...",
      "initiative_id":  "b4a1e2c3-...",
      "name":           "Core Development",
      "amount_in_cents": 10000000,
      "sort_order":     0
    }
  ],
  "financials": {
    "total_raised_cents":       0,
    "total_disbursed_cents":    0,
    "available_balance_cents":  0,
    "supporters":               0,
    "goals_total_cents":        10000000,
    "funded_percent":           0
  },
  "sponsors":    [],
  "created_on":  "2026-05-25T14:00:00Z",
  "updated_on":  "2026-05-25T14:00:00Z"
}
```

At this point the reviewer inbox receives an email with:
- **View** → `https://crowdfunding.linuxfoundation.org/initiatives/my-awesome-project`
- **Approve** → `https://crowdfunding.linuxfoundation.org/initiatives/my-awesome-project/process-approval/approve`
- **Decline** → `https://crowdfunding.linuxfoundation.org/initiatives/my-awesome-project/process-approval/decline`

### Step 2 — Reviewer clicks "Approve"

Browser opens: `https://crowdfunding.linuxfoundation.org/initiatives/my-awesome-project/process-approval/approve`

1. Frontend loads the page → checks reviewer is logged in (redirect to Auth0 login if not).
2. Frontend fetches `/v1/initiatives/my-awesome-project` → gets `id = "b4a1e2c3-..."`.
3. Frontend calls:

```http
POST /v1/initiatives/b4a1e2c3-dead-beef-1234-56789abcdef0/process-approval/approve
Authorization: Bearer eyJhbGciOiJSUzI1Ni...  (reviewer's token)
```

**Response `200 OK`:**

```json
{
  "id":     "b4a1e2c3-dead-beef-1234-56789abcdef0",
  "status": "published",
  ...
}
```

4. The initiative owner receives a "Your initiative has been approved" email.
5. The initiative is now publicly visible via `GET /v1/initiatives/my-awesome-project`.

### Step 3 — Owner updates the logo after approval (PATCH)

```http
PATCH /v1/initiatives/b4a1e2c3-dead-beef-1234-56789abcdef0
Authorization: Bearer eyJhbGciOiJSUzI1Ni...  (owner's token)
Content-Type: application/json

{
  "logo_url": "https://cdn.example.com/new-logo.png"
}
```

Only `logo_url` is changed. All goals, description, and other fields are left exactly as they were.

**Response `200 OK`:** returns the full updated initiative with the new `logo_url`.
