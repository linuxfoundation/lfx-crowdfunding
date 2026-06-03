# Logo Upload — Frontend Integration Guide

> **Audience:** Frontend engineers who are new to this codebase or new to S3 presigned URLs.
> This guide explains the concept from scratch, shows every file you need to create or edit,
> and ends with a manual testing checklist.

---

## Why presigned URLs exist

Uploading a file through the backend API is wasteful: the file travels from the browser → Nuxt server → Go API → S3. That doubles bandwidth and makes the API responsible for large binary payloads.

**Presigned URLs solve this.** The flow is:

1. Browser asks the backend *"give me a short-lived upload token"*
2. Backend generates a signed S3 URL and returns it immediately (no binary data involved)
3. Browser uploads the file **directly to S3** using that token
4. Browser saves the resulting permanent S3 URL in the initiative form

The API never touches the binary file at all.

```
Browser ──POST /api/presigned-url──► Nuxt BFF ──POST /v1/presigned-url──► Go API ──► S3 (signs URL)
                                                                                         │
                                     ◄── { upload_url, destination_url } ───────────────┘
                                                         │
Browser ──PUT <upload_url> (binary)────────────────────────────────────────────────────► S3
                                     ◄── 200 OK ─────────────────────────────────────────┘
                                                         │
Browser saves destination_url as logo_url in the initiative form
```

---

## Backend endpoint reference

### `POST /v1/presigned-url`

**Auth:** JWT required (send via the Nuxt BFF — see Step 1 below)

**Request body:**
```json
{ "content_type": "image/png" }
```

**Allowed content types:**

| Value | Format |
|-------|--------|
| `image/png` | PNG |
| `image/jpeg` | JPEG / JPG |
| `image/gif` | Animated or static GIF |
| `image/webp` | WebP |

> SVG is intentionally blocked — SVG files can contain JavaScript and S3 would serve them
> executable, creating a stored-XSS risk.

**Success response `200`:**
```json
{
  "upload_url": "https://lfx-logos.s3.us-east-1.amazonaws.com/a1b2c3...?X-Amz-Signature=...",
  "destination_url": "https://lfx-logos.s3.us-east-1.amazonaws.com/a1b2c3...",
  "required_headers": {
    "Content-Type": "image/png",
    "x-amz-acl": "public-read"
  }
}
```

| Field | Description |
|-------|-------------|
| `upload_url` | Presigned PUT URL. Valid for **3 minutes**. Use it once then discard. |
| `destination_url` | Permanent public URL. This is what you store as `logo_url` on the initiative. |
| `required_headers` | HTTP headers the client **must** include verbatim on the PUT request. They are part of the presigned URL signature — omitting any one causes S3 to return `403 SignatureDoesNotMatch`. |

**Error responses:**

| Status | When |
|--------|------|
| `400` | Missing or disallowed `content_type` |
| `401` | No valid JWT |
| `500` | AWS credentials not configured / S3 unreachable |

---

## Step 1 — Add the Nuxt BFF proxy

The frontend never calls the Go API directly. All authenticated calls go through a
server-side Nuxt route that attaches the user's JWT from the HTTP-only session cookie.

Create **`frontend/server/api/presigned-url.post.ts`**:

```typescript
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody } from 'h3'
import { useBackendFetch } from '../utils/backend-fetch'

interface PresignedURLWire {
  upload_url: string
  destination_url: string
  required_headers: Record<string, string>
}

export interface PresignedURLResult {
  uploadUrl: string
  destinationUrl: string
  requiredHeaders: Record<string, string>
}

export default defineEventHandler(async (event): Promise<PresignedURLResult> => {
  const body = await readBody<{ contentType: string }>(event)

  const raw = await useBackendFetch<PresignedURLWire>(event, '/v1/presigned-url', {
    method: 'POST',
    body: { content_type: body.contentType },
  })

  return {
    uploadUrl: raw.upload_url,
    destinationUrl: raw.destination_url,
    requiredHeaders: raw.required_headers,
  }
})
```

`useBackendFetch` automatically injects the `Authorization: Bearer <token>` header from
the user's session cookie. You do not need to touch auth at all.

---

## Step 2 — Add the shared type

Add to **`frontend/shared/types/upload.types.ts`** (create if it doesn't exist):

```typescript
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface PresignedURLResult {
  uploadUrl: string
  destinationUrl: string
  requiredHeaders: Record<string, string>
}

export type AllowedLogoMimeType = 'image/png' | 'image/jpeg' | 'image/gif' | 'image/webp'

export const ALLOWED_LOGO_MIME_TYPES: AllowedLogoMimeType[] = [
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/webp',
]

/** Maximum logo file size accepted (2 MB). Enforced client-side only — S3 does not enforce this. */
export const MAX_LOGO_SIZE_BYTES = 2 * 1024 * 1024
```

---

## Step 3 — Write the composable

Create **`frontend/app/composables/useLogoUpload.ts`**:

```typescript
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref } from 'vue'
import { ALLOWED_LOGO_MIME_TYPES, MAX_LOGO_SIZE_BYTES } from '#shared/types/upload.types'
import type { AllowedLogoMimeType } from '#shared/types/upload.types'

export const useLogoUpload = () => {
  const uploading = ref(false)
  const error = ref<string | null>(null)

  /**
   * Validates, requests a presigned URL, uploads the file directly to S3,
   * and returns the permanent public destination URL.
   *
   * @param file - The File object from an <input type="file"> change event.
   * @returns The permanent S3 URL to store as logo_url, or null on failure.
   */
  const uploadLogo = async (file: File): Promise<string | null> => {
    uploading.value = true
    error.value = null

    try {
      // 1. Client-side validation (mirrors the backend allowlist)
      if (!ALLOWED_LOGO_MIME_TYPES.includes(file.type as AllowedLogoMimeType)) {
        error.value = `Unsupported file type "${file.type}". Use PNG, JPEG, GIF, or WebP.`
        return null
      }
      if (file.size > MAX_LOGO_SIZE_BYTES) {
        error.value = `File is too large (${(file.size / 1024 / 1024).toFixed(1)} MB). Maximum is 2 MB.`
        return null
      }

      // 2. Ask the BFF for a presigned URL
      const { uploadUrl, destinationUrl, requiredHeaders } = await $fetch<{
        uploadUrl: string
        destinationUrl: string
        requiredHeaders: Record<string, string>
      }>(
        '/api/presigned-url',
        {
          method: 'POST',
          body: { contentType: file.type },
        }
      )

      // 3. PUT the file directly to S3 — NOT through the backend.
      //    requiredHeaders contains the headers that were signed into the presigned URL
      //    (e.g. Content-Type, x-amz-acl). They MUST be sent verbatim or S3 returns 403.
      const s3Response = await fetch(uploadUrl, {
        method: 'PUT',
        headers: requiredHeaders,
        body: file,
      })

      if (!s3Response.ok) {
        error.value = `Upload to S3 failed (HTTP ${s3Response.status}). The presigned URL may have expired — try again.`
        return null
      }

      // 4. Return the permanent URL for the caller to store on the initiative
      return destinationUrl
    } catch (e: unknown) {
      const err = e as { data?: { error?: string }; message?: string }
      error.value =
        err?.data?.error ?? err?.message ?? 'Logo upload failed. Please try again.'
      return null
    } finally {
      uploading.value = false
    }
  }

  return { uploading, error, uploadLogo }
}
```

### Why `fetch` instead of `$fetch` for the S3 PUT?

`$fetch` (ofetch) automatically sets `Content-Type: application/json` and serialises the
body as JSON. The S3 presigned PUT requires the raw binary and the exact content-type that
was signed. Use the browser's native `fetch` for this call only.

---

## Step 4 — Use it in a component

Here is a minimal example. Adapt it to your existing form component:

```vue
<script setup lang="ts">
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref } from 'vue'
import { ALLOWED_LOGO_MIME_TYPES } from '#shared/types/upload.types'

const { uploading, error, uploadLogo } = useLogoUpload()

// logoUrl is the value you eventually send as `logo_url` in the create/update payload
const logoUrl = ref<string | null>(null)
const previewSrc = ref<string | null>(null)

async function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  // Show a local preview immediately (doesn't require the upload to succeed)
  previewSrc.value = URL.createObjectURL(file)

  const url = await uploadLogo(file)
  if (url) {
    logoUrl.value = url
  }
}
</script>

<template>
  <div>
    <img v-if="previewSrc" :src="previewSrc" alt="Logo preview" class="h-16 w-16 object-contain" />

    <label class="cursor-pointer">
      <span>{{ uploading ? 'Uploading…' : 'Choose logo' }}</span>
      <input
        type="file"
        class="sr-only"
        :accept="ALLOWED_LOGO_MIME_TYPES.join(',')"
        :disabled="uploading"
        @change="onFileSelected"
      />
    </label>

    <p v-if="error" class="text-red-600 text-sm">{{ error }}</p>
    <p v-if="logoUrl" class="text-green-600 text-sm">Uploaded ✓</p>
  </div>
</template>
```

Then, when the user submits the form, include `logoUrl.value` in the payload:

```typescript
await $fetch('/api/initiatives', {
  method: 'POST',
  body: {
    name: form.name,
    // ...other fields...
    logo_url: logoUrl.value ?? undefined,
  },
})
```

---

## Step 5 — Infrastructure checklist (ask DevOps / platform team)

The backend code is done. Before this works end-to-end in any environment, the bucket
needs two things:

### CORS policy

Without CORS, the browser's direct `PUT` to S3 will be blocked. The bucket needs a
CORS rule that allows `PUT` from the frontend origin:

```json
[
  {
    "AllowedHeaders": ["Content-Type", "x-amz-acl"],
    "AllowedMethods": ["PUT"],
    "AllowedOrigins": ["https://crowdfunding.linuxfoundation.org"],
    "ExposeHeaders": []
  }
]
```

For local dev, add `http://localhost:3000` to `AllowedOrigins`.

### ACLs enabled

The upload sets `x-amz-acl: public-read` on every object so the `destination_url` is
publicly accessible without any further authentication. This requires:

- **Object Ownership** on the bucket set to `BucketOwnerPreferred` or `ObjectWriter`
- Block Public Access settings that permit ACLs (i.e. `BlockPublicAcls` = false)

If your bucket enforces Block Public Access instead, switch to a **bucket policy** that
grants public `s3:GetObject` on all objects, and remove the ACL from the `PutObjectInput`
in `s3_client.go`.

---

## Manual testing checklist

1. **Happy path — PNG upload**
   - Pick a valid PNG < 2 MB
   - Verify the component shows the local preview immediately
   - Verify `uploading` is `true` during the upload and `false` after
   - Verify `logoUrl` is set to a URL starting with `https://`
   - Open `logoUrl` in a new tab — image should load without auth

2. **Wrong file type**
   - Pick an `.svg` or `.pdf`
   - Verify the error message mentions the rejected type
   - Verify no network request is made to `/api/presigned-url`

3. **Oversized file**
   - Pick any image > 2 MB
   - Verify the error message shows the file size
   - Verify no network request is made to `/api/presigned-url`

4. **Expired presigned URL (simulate)**
   - In `s3_client.go`, temporarily change `defaultPresignExpiry` to `1 * time.Second`
   - Start the upload, wait 2 seconds, then let it proceed (or use DevTools to delay the PUT)
   - Verify S3 returns `403` and the composable surfaces the "may have expired" error

5. **Unauthenticated request**
   - Sign out (clear the session cookie)
   - Try to upload
   - Verify the BFF returns `401` and the composable surfaces an error message

6. **Form submission**
   - Complete a successful upload
   - Submit the initiative create/update form
   - Verify the saved initiative's `logo_url` matches the `destination_url` returned by S3
