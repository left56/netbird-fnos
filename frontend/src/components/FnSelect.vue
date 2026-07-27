<script setup lang="ts">
defineProps<{ modelValue?: string }>();
defineEmits(["update:modelValue"]);
</script>
<template>
  <select
    class="select"
    :value="modelValue"
    @change="
      $emit('update:modelValue', ($event.target as HTMLSelectElement).value)
    "
  >
    <slot />
  </select>
</template>
<style scoped>
.select {
  width: 100%;
  height: var(--fn-control);
  padding: 0 10px;
  border: 1px solid var(--fn-border);
  border-radius: 9px;
  background: #fff;
}
</style>
