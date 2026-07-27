<template>
  <div class="table-wrap">
    <table>
      <slot />
    </table>
  </div>
</template>
<style scoped>
.table-wrap {
  overflow: auto;
  border: 1px solid var(--fn-border);
  border-radius: 11px;
}
table {
  width: 100%;
  border-collapse: collapse;
}
:slotted(th) {
  background: #f8fafc;
  color: var(--fn-muted);
  font-size: 12px;
}
:slotted(th),
:slotted(td) {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid var(--fn-border);
}
</style>
