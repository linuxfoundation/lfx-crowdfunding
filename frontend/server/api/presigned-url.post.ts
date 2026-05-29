// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody } from 'h3';
import { useBackendFetch } from '../utils/backend-fetch';
import type { PresignedURLWire } from '../types/upload.types';
import type { PresignedURLResult } from '#shared/types/upload.types';

export default defineEventHandler(async (event): Promise<PresignedURLResult> => {
  const body = await readBody<{ contentType: string }>(event);

  const raw = await useBackendFetch<PresignedURLWire>(event, '/v1/presigned-url', {
    method: 'POST',
    body: { content_type: body.contentType },
  });

  return {
    uploadUrl: raw.upload_url,
    destinationUrl: raw.destination_url,
    requiredHeaders: raw.required_headers,
  };
});
