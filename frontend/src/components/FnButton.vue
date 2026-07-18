<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: "primary" | "default" | "danger";
    disabled?: boolean;
  }>(),
  { variant: "default" },
);
</script>
<template>
  <button class="button" :class="variant" :disabled="disabled"><slot /></button>
</template>
<style scoped>
.button {
  height: var(--fn-control);
  padding: 0 15px;
  border: 1px solid var(--fn-border);
  border-radius: 9px;
  background: #fff;
  color: var(--fn-text);
}
.button.primary {
  background: var(--fn-primary);
  border-color: var(--fn-primary);
  color: #fff;
}
.button.danger {
  color: #d92d20;
  border-color: #f5c2be;
  background: #fff7f6;
}
</style>
