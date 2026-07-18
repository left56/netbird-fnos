<script setup lang="ts">
defineProps<{
  modelValue?: string;
  placeholder?: string;
  type?: string;
  disabled?: boolean;
}>();
defineEmits(["update:modelValue"]);
</script>
<template>
  <input
    class="input"
    :type="type || 'text'"
    :placeholder="placeholder"
    :disabled="disabled"
    :value="modelValue"
    @input="
      $emit('update:modelValue', ($event.target as HTMLInputElement).value)
    "
  />
</template>
<style scoped>
.input {
  width: 100%;
  height: var(--fn-control);
  padding: 0 11px;
  border: 1px solid var(--fn-border);
  border-radius: 9px;
  background: #fff;
  outline: none;
}
.input:focus {
  border-color: var(--fn-primary);
  box-shadow: 0 0 0 3px #e8f1ff;
}
</style>
