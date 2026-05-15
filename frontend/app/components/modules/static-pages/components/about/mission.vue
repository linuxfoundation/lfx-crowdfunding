<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 flex gap-8 items-start md:flex-row flex-col">
    <!-- Left sidebar -->
    <div class="md:basis-1/3 flex items-start md:py-16 pt-10">
      <div class="flex items-center gap-5">
        <img
          :src="bullet"
          alt=""
          class="h-8 w-auto"
        />
        <h2 class="text-xl font-semibold leading-8 text-neutral-900 whitespace-nowrap">Our Mission</h2>
      </div>
    </div>

    <!-- Right content -->
    <div class="md:basis-2/3 flex flex-col gap-6 md:py-16 pb-16">
      <p class="text-base font-normal leading-6 text-neutral-900">
        Our mission is to promote a sustainable and healthy open source ecosystem and work for the common interests of
        those who create and use open source software.
      </p>
      <p class="text-base font-normal leading-6 text-neutral-900">
        LFX CrowdFunding is operated by the Linux Foundation, a 501(c)(6) nonprofit that has been providing financial,
        legal, and operational infrastructure to open source projects for over two decades. We handle the
        behind-the-scenes work — issuing invoices, managing funds, contract review, compliance — so maintainers can
        focus on building great software.
      </p>
      <p class="text-base font-normal leading-6 text-neutral-900">
        Our team has deep experience in both nonprofit operations and the open source community. This dual expertise
        allows us to understand and meet the unique needs of open source projects at every stage — from individual
        maintainers to foundations supporting thousands of contributors.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import bullet from '~/assets/images/about-image-bullet.png';
</script>

<script lang="ts">
export default {
  name: 'AboutMission',
};
</script>
