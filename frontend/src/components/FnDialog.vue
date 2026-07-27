<script setup lang="ts">
defineProps<{ open: boolean; title: string }>();
defineEmits(["close"]);
</script>
<template>
  <div v-if="open" class="mask" @click.self="$emit('close')">
    <section class="dialog">
      <header>
        <h2>{{ title }}</h2>
        <button @click="$emit('close')">×</button>
      </header>
      <div class="body"><slot /></div>
      <footer><slot name="footer" /></footer>
    </section>
  </div>
</template>
<style scoped>
.mask {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  background: #10182866;
  z-index: 10;
}
.dialog {
  width: min(560px, calc(100vw - 32px));
  background: #fff;
  border-radius: 15px;
  overflow: hidden;
  box-shadow: 0 20px 45px #10182833;
}
.dialog header,
.dialog footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--fn-border);
}
.dialog footer {
  border-bottom: 0;
  border-top: 1px solid var(--fn-border);
  justify-content: flex-end;
  gap: 8px;
}
.dialog h2 {
  margin: 0;
  font-size: 17px;
}
.dialog header button {
  border: 0;
  background: transparent;
  font-size: 24px;
}
.body {
  padding: 20px;
}
</style>
