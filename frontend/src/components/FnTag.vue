<script setup lang="ts">
withDefaults(
  defineProps<{ type?: "primary" | "success" | "warning" | "muted" }>(),
  { type: "muted" },
);
</script>
<template>
  <span class="tag" :class="type"><slot /></span>
</template>
<style scoped>
.tag {
  display: inline-flex;
  padding: 3px 8px;
  border-radius: 99px;
  font-size: 12px;
  background: #f1f5f9;
  color: #64748b;
}
.primary {
  background: #eaf3ff;
  color: #2678e8;
}
.success {
  background: #eaf8ef;
  color: #16803c;
}
.warning {
  background: #fff6dd;
  color: #9a6700;
}
</style>
