<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="min-h-screen flex flex-col">
    <!-- LFX header web component (rendered client-side only) -->
    <ClientOnly>
      <lfx-header app-name="Crowdfunding" />
    </ClientOnly>

    <main class="flex-grow">
      <slot />
    </main>

    <footer class="py-6 px-4 border-t border-gray-200 text-sm text-gray-500 text-center">
      © {{ new Date().getFullYear() }} The Linux Foundation. All rights reserved.
    </footer>
  </div>
</template>
