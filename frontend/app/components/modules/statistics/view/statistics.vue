<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="bg-white">
    <div class="container">
      <!-- Hero header -->
      <div class="md:pt-21 pt-10">
        <statistics-header
          :overview="overviewData"
          :is-loading="overviewLoading"
        />
      </div>

      <!-- Body -->
      <div class="md:pb-30 pb-20">
        <div class="flex flex-col lg:flex-row gap-8 items-start">
          <!-- Left column -->
          <div class="flex-1 min-w-0 flex flex-col md:gap-6 gap-8 w-full">
            <statistics-funding-by-category
              :categories="categoryData?.data ?? []"
              :is-loading="categoryLoading"
              :error="categoryError"
            />

            <statistics-donor-breakdown
              :breakdown="breakdownData"
              :is-loading="breakdownLoading"
              :error="breakdownError"
            />

            <statistics-monthly-donations
              :monthly="monthlyData"
              :is-loading="monthlyLoading"
              :error="monthlyError"
            />

            <statistics-top-organizations
              :entries="topDonorsData?.organizations ?? []"
              :is-loading="topDonorsLoading"
              :error="topDonorsError"
            />

            <statistics-top-individuals
              :entries="topDonorsData?.individuals ?? []"
              :is-loading="topDonorsLoading"
              :error="topDonorsError"
            />
          </div>

          <!-- Right column -->
          <div class="w-full lg:w-[340px] shrink-0 md:sticky md:top-27 relative">
            <statistics-recent-donations />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import StatisticsHeader from '../components/statistics-header.vue';
import StatisticsFundingByCategory from '../components/statistics-funding-by-category.vue';
import StatisticsDonorBreakdown from '../components/statistics-donor-breakdown.vue';
import StatisticsMonthlyDonations from '../components/statistics-monthly-donations.vue';
import StatisticsTopOrganizations from '../components/statistics-top-organizations.vue';
import StatisticsTopIndividuals from '../components/statistics-top-individuals.vue';
import StatisticsRecentDonations from '../components/statistics-recent-donations.vue';
import { useStatisticsOverview } from '~/composables/statistics/useStatisticsOverview';
import { useStatisticsFundingByCategory } from '~/composables/statistics/useStatisticsFundingByCategory';
import { useStatisticsDonorBreakdown } from '~/composables/statistics/useStatisticsDonorBreakdown';
import { useStatisticsMonthlyDonations } from '~/composables/statistics/useStatisticsMonthlyDonations';
import { useStatisticsTopDonors } from '~/composables/statistics/useStatisticsTopDonors';

const { data: overviewData, isLoading: overviewLoading } = useStatisticsOverview();

const { data: categoryData, isLoading: categoryLoading, error: categoryRawError } = useStatisticsFundingByCategory();
const categoryError = computed(() => categoryRawError.value as Error | null);

const { data: breakdownData, isLoading: breakdownLoading, error: breakdownRawError } = useStatisticsDonorBreakdown();
const breakdownError = computed(() => breakdownRawError.value as Error | null);

const { data: monthlyData, isLoading: monthlyLoading, error: monthlyRawError } = useStatisticsMonthlyDonations();
const monthlyError = computed(() => monthlyRawError.value as Error | null);

const { data: topDonorsData, isLoading: topDonorsLoading, error: topDonorsRawError } = useStatisticsTopDonors();
const topDonorsError = computed(() => topDonorsRawError.value as Error | null);
</script>

<script lang="ts">
export default {
  name: 'StatisticsView',
};
</script>
