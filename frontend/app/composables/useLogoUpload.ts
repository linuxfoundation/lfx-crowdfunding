// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref } from 'vue';
import { ALLOWED_LOGO_MIME_TYPES, MAX_LOGO_SIZE_BYTES } from '#shared/types/upload.types';
import type { AllowedLogoMimeType, PresignedURLResult } from '#shared/types/upload.types';

export const useLogoUpload = () => {
  const uploading = ref(false);
  const error = ref<string | null>(null);

  const uploadLogo = async (file: File): Promise<string | null> => {
    uploading.value = true;
    error.value = null;

    try {
      if (!ALLOWED_LOGO_MIME_TYPES.includes(file.type as AllowedLogoMimeType)) {
        error.value = `Unsupported file type "${file.type}". Use PNG, JPEG, GIF, or WebP.`;
        return null;
      }
      if (file.size > MAX_LOGO_SIZE_BYTES) {
        error.value = `File is too large (${(file.size / 1024 / 1024).toFixed(1)} MB). Maximum is 2 MB.`;
        return null;
      }

      const { uploadUrl, destinationUrl, requiredHeaders } = await $fetch<PresignedURLResult>(
        '/api/presigned-url',
        {
          method: 'POST',
          body: { contentType: file.type },
        },
      );

      // Use native fetch — $fetch would JSON-encode the binary body and override Content-Type.
      const s3Response = await fetch(uploadUrl, {
        method: 'PUT',
        headers: requiredHeaders,
        body: file,
      });

      if (!s3Response.ok) {
        error.value = `Upload to S3 failed (HTTP ${s3Response.status}). The presigned URL may have expired — try again.`;
        return null;
      }

      return destinationUrl;
    } catch (e: unknown) {
      const err = e as { data?: { error?: string }; message?: string };
      error.value = err?.data?.error ?? err?.message ?? 'Logo upload failed. Please try again.';
      return null;
    } finally {
      uploading.value = false;
    }
  };

  return { uploading, error, uploadLogo };
};
