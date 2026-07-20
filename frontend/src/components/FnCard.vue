<template>
  <section class="card"><slot /></section>
</template>
<style scoped>
.card {
  background: var(--fn-panel);
  border: 1px solid var(--fn-border);
  border-radius: var(--fn-radius);
  padding: 20px;
  box-shadow: 0 2px 10px rgb(31 41 55 / 3%);
}
</style>
