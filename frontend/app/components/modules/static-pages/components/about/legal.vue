<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="py-16">
    <div class="bg-neutral-50 rounded-xl p-10 flex flex-col gap-8">
      <div class="flex flex-col gap-2.5">
        <p class="text-base font-semibold leading-6 text-neutral-900">The Linux Foundation 501(c)(6)</p>
        <div class="flex flex-col text-base font-normal leading-6 text-neutral-600">
          <p>1 Letterman Drive, Building D, Suite D4700, San Francisco, CA 94129, United States</p>
          <p>
            Website:
            <a
              href="https://linuxfoundation.org/"
              target="_blank"
              class="text-accent-500"
              >linuxfoundation.org</a
            >
            | LFX Platform:
            <a
              href="https://lfx.linuxfoundation.org/"
              target="_blank"
              class="text-accent-500"
              >lfx.linuxfoundation.org</a
            >
          </p>
        </div>
      </div>

      <p class="text-base font-normal leading-6 text-neutral-500 italic">
        Contributions to initiatives hosted on LFX CrowdFunding are not deductible as charitable contributions for U.S.
        federal income tax purposes. The Linux Foundation is a 501(c)(6) trade association.
      </p>
    </div>
  </div>
</template>

<script lang="ts">
export default {
  name: 'AboutLegal',
};
</script>
