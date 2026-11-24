<script setup lang="ts">
defineProps<{
  completed: boolean
}>()
</script>

<template>
  <svg width="34" height="34" viewBox="0 0 36 36" aria-hidden="true">
    <rect x="7" y="6" width="17" height="23" rx="3" fill="#ffffff" stroke="#1e293b" stroke-width="1.4" />
    <rect x="11" y="4" width="9" height="4" rx="1.5" fill="#93c5fd" />
    <line x1="11" y1="13" x2="20" y2="13" stroke="#3b82f6" stroke-width="2" stroke-linecap="round" />
    <line x1="11" y1="18" x2="20" y2="18" stroke="#64748b" stroke-width="2" stroke-linecap="round" />
    <line x1="11" y1="23" x2="17" y2="23" stroke="#64748b" stroke-width="2" stroke-linecap="round" />
    <path d="M22 25 L30 13 L33 15 L25 27 L21 28 Z" fill="#facc15" stroke="#92400e" stroke-width="1.2" />
  </svg>
  <span v-if="completed" class="pin-done-badge" aria-label="Selesai">
    <svg width="12" height="12" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <path d="M3.5 8.5L6.5 11.5L12.5 5" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
  </span>
</template>
