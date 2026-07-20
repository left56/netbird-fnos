<script setup lang="ts">
defineProps<{ title: string; description?: string }>();
</script>
<template>
  <header>
    <div>
      <h1>{{ title }}</h1>
      <p v-if="description">{{ description }}</p>
    </div>
    <div class="actions"><slot /></div>
  </header>
</template>
<style scoped>
header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}
h1 {
  margin: 0;
  font-size: 22px;
}
p {
  margin: 7px 0 0;
  color: var(--fn-muted);
  font-size: 14px;
}
.actions {
  display: flex;
  gap: 8px;
}
</style>
