<script setup lang="ts">
defineProps<{ label: string; hint?: string }>();
</script>
<template>
  <label class="item"
    ><span>{{ label }}</span
    ><slot /><small v-if="hint">{{ hint }}</small></label
  >
</template>
<style scoped>
.item {
  display: grid;
  gap: 7px;
  font-size: 14px;
  font-weight: 600;
}
.item small {
  font-weight: 400;
  color: var(--fn-muted);
}
</style>
