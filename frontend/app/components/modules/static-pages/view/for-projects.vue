<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="bg-white">
    <div class="container">
      <div class="md:pt-18 pt-10 md:px-35 px-0">
        <for-projects-hero />
        <for-projects-stats
          :overview="overviewData"
          :is-loading="overviewLoading"
        />
        <for-projects-fund-types />
        <for-projects-how-it-works />
        <for-projects-why-choose />
        <for-projects-testimonial />
        <for-projects-featured-initiatives
          :initiatives="initiativesData?.data ?? []"
          :is-loading="initiativesLoading"
          :error="initiativesError"
        />
        <for-projects-cta />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ForProjectsHero from '../components/for-projects/hero.vue';
import ForProjectsStats from '../components/for-projects/stats.vue';
import ForProjectsFundTypes from '../components/for-projects/fund-types.vue';
import ForProjectsHowItWorks from '../components/for-projects/how-it-works.vue';
import ForProjectsWhyChoose from '../components/for-projects/why-choose.vue';
import ForProjectsTestimonial from '../components/for-projects/testimonial.vue';
import ForProjectsFeaturedInitiatives from '../components/for-projects/featured-initiatives.vue';
import ForProjectsCta from '../components/for-projects/cta.vue';
import { useStatisticsOverview } from '~/composables/statistics/useStatisticsOverview';
import { useFeaturedInitiatives } from '~/composables/static-pages/useFeaturedInitiatives';

const { data: overviewData, isLoading: overviewLoading } = useStatisticsOverview();

const { data: initiativesData, isLoading: initiativesLoading, error: initiativesRawError } = useFeaturedInitiatives();
const initiativesError = computed(() => initiativesRawError.value as Error | null);
</script>

<script lang="ts">
export default {
  name: 'ForProjectsView',
};
</script>
