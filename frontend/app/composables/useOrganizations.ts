// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref } from 'vue';
import type { Organization } from '#shared/types/organization.types';

const organizations = ref<Organization[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
let fetchInFlight: Promise<void> | null = null;

export const useOrganizations = () => {
  const fetchOrganizations = async () => {
    if (fetchInFlight) return fetchInFlight;
    fetchInFlight = (async () => {
      loading.value = true;
      error.value = null;
      try {
        organizations.value = await $fetch<Organization[]>('/api/me/organizations');
      } catch (e: unknown) {
        const err = e as { message?: string };
        error.value = err?.message ?? 'Could not load your organizations.';
      } finally {
        loading.value = false;
        fetchInFlight = null;
      }
    })();
    return fetchInFlight;
  };

  const createOrganization = async (
    name: string,
    avatarUrl: string,
  ): Promise<Organization | null> => {
    try {
      const org = await $fetch<Organization>('/api/me/organizations', {
        method: 'POST',
        body: { name, avatar_url: avatarUrl },
      });
      organizations.value = [...organizations.value, org];
      return org;
    } catch (e: unknown) {
      const err = e as { message?: string };
      error.value = err?.message ?? 'Could not create organization.';
      return null;
    }
  };

  const updateOrganization = async (
    id: string,
    name: string,
    avatarUrl: string,
  ): Promise<Organization | null> => {
    try {
      const org = await $fetch<Organization>(`/api/me/organizations/${id}`, {
        method: 'PATCH',
        body: { name, avatar_url: avatarUrl },
      });
      organizations.value = organizations.value.map((o) => (o.id === id ? org : o));
      return org;
    } catch (e: unknown) {
      const err = e as { message?: string };
      error.value = err?.message ?? 'Could not update organization.';
      return null;
    }
  };

  const deleteOrganization = async (id: string): Promise<boolean> => {
    try {
      await $fetch(`/api/me/organizations/${id}`, { method: 'DELETE' });
      organizations.value = organizations.value.filter((o) => o.id !== id);
      return true;
    } catch (e: unknown) {
      const err = e as { message?: string };
      error.value = err?.message ?? 'Could not delete organization.';
      return false;
    }
  };

  return {
    organizations,
    loading,
    error,
    fetchOrganizations,
    createOrganization,
    updateOrganization,
    deleteOrganization,
  };
};
