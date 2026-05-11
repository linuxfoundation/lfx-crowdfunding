<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div>
    <crowdfunding-hero />

    <!-- Campaign list -->
    <section class="container py-16">
      <h2 class="text-heading-2 font-bold mb-8">Active Campaigns</h2>

      <!-- Loading skeletons -->
      <div
        v-if="isLoading"
        class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"
      >
        <lfx-card
          v-for="n in 3"
          :key="n"
        >
          <div class="p-6 flex flex-col gap-4">
            <lfx-skeleton
              width="4rem"
              height="1.5rem"
              :rounded="true"
            />
            <lfx-skeleton height="1.25rem" />
            <div class="flex flex-col gap-2">
              <lfx-skeleton height="0.75rem" />
              <lfx-skeleton
                width="75%"
                height="0.75rem"
              />
            </div>
            <lfx-skeleton height="0.5rem" />
            <lfx-skeleton
              width="50%"
              height="1rem"
            />
          </div>
        </lfx-card>
      </div>

      <!-- Error -->
      <div
        v-else-if="error"
        class="flex items-center gap-2 text-negative-600"
      >
        <lfx-icon
          name="circle-exclamation"
          type="solid"
          :size="16"
        />
        <span class="text-body-1">Failed to load campaigns. Please try again.</span>
      </div>

      <!-- Entity cards -->
      <div
        v-else-if="data"
        class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"
      >
        <lfx-card
          v-for="entity in data.data"
          :key="entity.id"
        >
          <div class="p-6 flex flex-col gap-4">
            <lfx-tag
              :variation="tagVariation(entity.entityType)"
              size="small"
              type="transparent"
            >
              {{ entityTypeLabel(entity.entityType) }}
            </lfx-tag>

            <h3 class="text-heading-4 font-semibold">{{ entity.name }}</h3>

            <p class="text-body-1 text-neutral-600 line-clamp-2 flex-grow">
              {{ entity.description }}
            </p>

            <div class="flex flex-col gap-2">
              <lfx-progress-bar
                :values="[progressPercent(entity)]"
                color="normal"
                size="small"
              />
              <p class="text-body-2 text-neutral-500">
                <span class="font-semibold text-neutral-900">
                  {{ formatAmount(entity.fundingStatus?.totalDonationsInCents ?? 0) }}
                </span>
                raised of {{ formatAmount(entity.fundingStatus?.totalAnnualGoalInCents ?? 0) }}
              </p>
            </div>

            <lfx-button
              label="Donate"
              type="primary"
              button-style="rounded"
              size="small"
            />
          </div>
        </lfx-card>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useEntities } from '~/composables/useEntities';
import type { Entity } from '~/types/entity.types';
import type { TagStyle } from '~/components/uikit/tag/types/tag.types';
import CrowdfundingHero from '~/components/shared/hero.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxTag from '~/components/uikit/tag/tag.vue';
import LfxProgressBar from '~/components/uikit/progress-bar/progress-bar.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';

useHead({ title: 'Home' });

const { data, isLoading, error } = useEntities();

const tagVariation = (entityType: string): TagStyle => {
  const map: Record<string, TagStyle> = {
    project: 'info',
    mentorship: 'positive',
    general_fund: 'neutral',
    event: 'warning',
  };
  return map[entityType] ?? 'neutral';
};

const entityTypeLabel = (entityType: string): string =>
  entityType.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());

const formatAmount = (cents: number): string => '$' + (cents / 100).toLocaleString();

const progressPercent = (entity: Entity): number => {
  const goal = entity.fundingStatus?.totalAnnualGoalInCents ?? 0;
  const raised = entity.fundingStatus?.totalDonationsInCents ?? 0;
  return goal > 0 ? Math.min(100, Math.round((raised / goal) * 100)) : 0;
};
</script>
