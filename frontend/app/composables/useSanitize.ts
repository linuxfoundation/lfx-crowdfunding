// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import DOMPurify from 'isomorphic-dompurify';

export const useSanitize = () => {
  const sanitize = (dirty: string) => DOMPurify.sanitize(dirty);

  return { sanitize };
};
