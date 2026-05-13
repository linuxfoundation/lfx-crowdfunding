<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="bg-white">
    <div class="container">
      <div class="px-35">
        <for-companies-hero />
        <for-companies-why-fund />
        <for-companies-portfolio />
        <for-companies-companies
          :companies="companiesData?.data ?? []"
          :is-loading="companiesLoading"
          :error="companiesError"
        />
        <for-companies-cta />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ForCompaniesHero from '../components/for-companies/hero.vue';
import ForCompaniesWhyFund from '../components/for-companies/why-fund.vue';
import ForCompaniesPortfolio from '../components/for-companies/portfolio.vue';
import ForCompaniesCompanies from '../components/for-companies/companies.vue';
import ForCompaniesCta from '../components/for-companies/cta.vue';
import { useInvestingCompanies } from '~/composables/static-pages/useInvestingCompanies';

const { data: companiesData, isLoading: companiesLoading, error: companiesRawError } = useInvestingCompanies();
const companiesError = computed(() => companiesRawError.value as Error | null);
</script>

<script lang="ts">
export default {
  name: 'ForCompaniesView',
};
</script>
