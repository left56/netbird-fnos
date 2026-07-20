<script setup lang="ts">
defineProps<{ modelValue: boolean; disabled?: boolean }>();
defineEmits(["update:modelValue"]);
</script>
<template>
  <button
    class="switch"
    :class="{ on: modelValue }"
    :disabled="disabled"
    @click="$emit('update:modelValue', !modelValue)"
  >
    <i />
  </button>
</template>
<style scoped>
.switch {
  width: 42px;
  height: 24px;
  border: 0;
  border-radius: 99px;
  background: #cbd5e1;
  padding: 3px;
}
.switch i {
  display: block;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #fff;
  transition: 0.15s;
}
.switch.on {
  background: var(--fn-primary);
}
.switch.on i {
  transform: translateX(18px);
}
</style>
