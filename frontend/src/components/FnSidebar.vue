<script setup lang="ts">
defineProps<{ items: { path: string; label: string }[] }>();
</script>
<template>
  <aside>
    <div class="brand">
      <span>◈</span><strong>NetBird</strong><small>fnOS</small>
    </div>
    <nav>
      <RouterLink v-for="item in items" :key="item.path" :to="item.path">{{
        item.label
      }}</RouterLink>
    </nav>
  </aside>
</template>
<style scoped>
aside {
  width: 232px;
  min-height: 100vh;
  padding: 22px 14px;
  background: var(--fn-sidebar);
  border-right: 1px solid var(--fn-border);
}
.brand {
  display: flex;
  align-items: baseline;
  gap: 7px;
  padding: 0 12px 22px;
}
.brand span {
  color: var(--fn-primary);
  font-size: 24px;
}
.brand small {
  color: var(--fn-muted);
}
nav {
  display: grid;
  gap: 4px;
}
a {
  padding: 11px 13px;
  border-radius: 10px;
  color: #556070;
  text-decoration: none;
  font-size: 14px;
}
a.router-link-exact-active {
  background: #fff;
  color: var(--fn-primary);
  font-weight: 600;
  box-shadow: 0 1px 4px #1018280d;
}
</style>
