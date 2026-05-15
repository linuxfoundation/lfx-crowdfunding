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
        <h2 class="text-xl font-semibold leading-8 text-neutral-900 whitespace-nowrap">Our Offer</h2>
      </div>
    </div>

    <!-- Right content -->
    <div class="md:basis-2/3 flex flex-col gap-8 md:py-16 pb-16">
      <div class="flex flex-col gap-3">
        <p class="text-base font-semibold leading-6 text-neutral-900">Financial Management</p>
        <p class="text-base font-normal leading-6 text-neutral-900">
          Comprehensive financial services including accounting, invoicing, and fund management. Through LFX
          CrowdFunding, your project can raise funds, pay contributors, and manage finances with full transparency —
          every donation and expense is publicly visible.
        </p>
      </div>

      <div class="flex flex-col gap-3">
        <p class="text-base font-semibold leading-6 text-neutral-900">Legal &amp; Compliance Support</p>
        <p class="text-base font-normal leading-6 text-neutral-900">
          We provide legal support including contract review and signature, ensuring your project stays compliant and
          protected. All funds flow through the Linux Foundation's established 501(c)(6) structure.
        </p>
      </div>

      <div class="flex flex-col gap-3">
        <p class="text-base font-semibold leading-6 text-neutral-900">Zero Platform Fees</p>
        <p class="text-base font-normal leading-6 text-neutral-900">
          We charge 0% fees. The Linux Foundation underwrites all platform and payment processing fees until we reach
          $10M in funding. Every dollar raised goes directly to your project. Funds are disbursed within 5 business
          days.
        </p>
      </div>

      <div class="flex flex-col gap-3">
        <p class="text-base font-semibold leading-6 text-neutral-900">Community &amp; Ecosystem</p>
        <p class="text-base font-normal leading-6 text-neutral-900">
          Join a network of projects backed by the Linux Foundation. Benefit from the trust and credibility that comes
          with LF stewardship, access to corporate sponsors, and a collaborative ecosystem of maintainers and
          contributors.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import bullet from '~/assets/images/about-image-bullet.png';
</script>

<script lang="ts">
export default {
  name: 'AboutOffer',
};
</script>
