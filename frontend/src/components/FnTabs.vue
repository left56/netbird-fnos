<script setup lang="ts">
defineProps<{ modelValue: string }>();
defineEmits(["update:modelValue"]);
</script>
<template>
  <nav class="tabs">
    <slot
      :select="(value: string) => $emit('update:modelValue', value)"
      :active="modelValue"
    />
  </nav>
</template>
<style scoped>
.tabs {
  display: flex;
  gap: 22px;
  border-bottom: 1px solid var(--fn-border);
  margin-bottom: 18px;
}
</style>
