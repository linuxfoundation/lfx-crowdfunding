<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="bg-white">
    <div class="container">
      <div class="md:pt-18 pt-10 md:px-35 px-0">
        <about-hero />
        <about-mission />
        <about-offer />
        <about-get-started />
        <about-legal />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import AboutHero from '../components/about/hero.vue';
import AboutMission from '../components/about/mission.vue';
import AboutOffer from '../components/about/offer.vue';
import AboutGetStarted from '../components/about/get-started.vue';
import AboutLegal from '../components/about/legal.vue';
</script>

<script lang="ts">
export default {
  name: 'AboutView',
};
</script>
