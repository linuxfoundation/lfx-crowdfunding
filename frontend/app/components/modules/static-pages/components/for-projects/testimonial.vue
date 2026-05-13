<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="py-16">
    <div
      v-for="testimonial in testimonials"
      :key="testimonial.id"
      class="bg-neutral-50 rounded-xl p-16 flex flex-col gap-10"
    >
      <p class="text-2xl font-light leading-9 text-neutral-900">"{{ testimonial.quote }}"</p>

      <div class="flex items-center gap-5">
        <div class="flex items-center gap-4 shrink-0">
          <img
            :src="testimonial.logoSrc"
            :alt="testimonial.logoAlt"
            class="h-8 w-auto"
          />
          <div class="h-6 w-px bg-neutral-200 shrink-0" />
        </div>
        <div class="flex flex-col">
          <p class="text-base font-semibold leading-6 text-neutral-900">{{ testimonial.authorName }}</p>
          <p class="text-sm font-normal leading-5 text-neutral-500">{{ testimonial.authorTitle }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { testimonials } from '~/config/pages/testimonials.config';
</script>

<script lang="ts">
export default {
  name: 'ForProjectsTestimonial',
};
</script>
