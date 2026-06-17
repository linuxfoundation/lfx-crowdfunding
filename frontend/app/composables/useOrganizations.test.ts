// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

const mockFetch = vi.fn();
globalThis.$fetch = mockFetch as typeof $fetch;

import { useOrganizations } from './useOrganizations';

const ORG_1 = {
  id: 'org-1',
  name: 'Acme',
  avatarUrl: 'https://example.com/logo.png',
  status: 'active',
};
const ORG_2 = {
  id: 'org-2',
  name: 'Beta',
  avatarUrl: 'https://example.com/beta.png',
  status: 'active',
};

describe('useOrganizations', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    const { organizations, loading, error } = useOrganizations();
    organizations.value = [];
    loading.value = false;
    error.value = null;
  });

  // ── fetchOrganizations ────────────────────────────────────────────────────

  describe('fetchOrganizations', () => {
    it('populates organizations on success', async () => {
      mockFetch.mockResolvedValueOnce([ORG_1, ORG_2]);

      const { fetchOrganizations, organizations } = useOrganizations();
      await fetchOrganizations();

      expect(organizations.value).toEqual([ORG_1, ORG_2]);
    });

    it('clears error before fetching', async () => {
      mockFetch.mockResolvedValueOnce([]);
      const { fetchOrganizations, error } = useOrganizations();
      error.value = 'prior error';

      await fetchOrganizations();

      expect(error.value).toBeNull();
    });

    it('sets error on failure', async () => {
      mockFetch.mockRejectedValueOnce({ message: 'network error' });

      const { fetchOrganizations, error } = useOrganizations();
      await fetchOrganizations();

      expect(error.value).toBe('network error');
    });

    it('deduplicates concurrent calls (fetchInFlight)', async () => {
      let resolve!: (v: (typeof ORG_1)[]) => void;
      mockFetch.mockReturnValueOnce(new Promise((r) => (resolve = r)));

      const { fetchOrganizations } = useOrganizations();
      const p1 = fetchOrganizations();
      const p2 = fetchOrganizations();

      resolve([ORG_1]);
      await Promise.all([p1, p2]);

      expect(mockFetch).toHaveBeenCalledTimes(1);
    });
  });

  // ── createOrganization ────────────────────────────────────────────────────

  describe('createOrganization', () => {
    it('appends the new org to the list and returns it', async () => {
      mockFetch.mockResolvedValueOnce(ORG_1);
      const { createOrganization, organizations } = useOrganizations();

      const result = await createOrganization('Acme', 'https://example.com/logo.png');

      expect(result).toEqual(ORG_1);
      expect(organizations.value).toContainEqual(ORG_1);
    });

    it('clears prior error before the attempt', async () => {
      mockFetch.mockResolvedValueOnce(ORG_1);
      const { createOrganization, error } = useOrganizations();
      error.value = 'prior error';

      await createOrganization('Acme', 'https://example.com/logo.png');

      expect(error.value).toBeNull();
    });

    it('sets error and returns null on failure', async () => {
      mockFetch.mockRejectedValueOnce({ message: 'create failed' });
      const { createOrganization, error } = useOrganizations();

      const result = await createOrganization('Acme', 'https://example.com/logo.png');

      expect(result).toBeNull();
      expect(error.value).toBe('create failed');
    });
  });

  // ── updateOrganization ────────────────────────────────────────────────────

  describe('updateOrganization', () => {
    it('replaces the matching org in the list and returns it', async () => {
      const updated = { ...ORG_1, name: 'Acme Updated' };
      mockFetch.mockResolvedValueOnce(updated);
      const { updateOrganization, organizations } = useOrganizations();
      organizations.value = [ORG_1, ORG_2];

      const result = await updateOrganization('org-1', 'Acme Updated', ORG_1.avatarUrl);

      expect(result).toEqual(updated);
      expect(organizations.value.find((o) => o.id === 'org-1')?.name).toBe('Acme Updated');
    });

    it('clears prior error before the attempt', async () => {
      mockFetch.mockResolvedValueOnce(ORG_1);
      const { updateOrganization, organizations, error } = useOrganizations();
      organizations.value = [ORG_1];
      error.value = 'prior error';

      await updateOrganization('org-1', 'Acme', ORG_1.avatarUrl);

      expect(error.value).toBeNull();
    });

    it('sets error and returns null on failure', async () => {
      mockFetch.mockRejectedValueOnce({ message: 'update failed' });
      const { updateOrganization, error } = useOrganizations();

      const result = await updateOrganization('org-1', 'Acme', 'https://example.com/logo.png');

      expect(result).toBeNull();
      expect(error.value).toBe('update failed');
    });
  });

  // ── deleteOrganization ────────────────────────────────────────────────────

  describe('deleteOrganization', () => {
    it('removes the org from the list and returns true', async () => {
      mockFetch.mockResolvedValueOnce(undefined);
      const { deleteOrganization, organizations } = useOrganizations();
      organizations.value = [ORG_1, ORG_2];

      const result = await deleteOrganization('org-1');

      expect(result).toBe(true);
      expect(organizations.value.map((o) => o.id)).toEqual(['org-2']);
    });

    it('sets error and returns false on failure', async () => {
      mockFetch.mockRejectedValueOnce({ message: 'delete failed' });
      const { deleteOrganization, error } = useOrganizations();

      const result = await deleteOrganization('org-1');

      expect(result).toBe(false);
      expect(error.value).toBe('delete failed');
    });
  });
});
