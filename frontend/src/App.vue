<script setup lang="ts">
import FnSidebar from "./components/FnSidebar.vue";
const links = [
  { path: "/", label: "概览", icon: "⌂" },
  { path: "/peers", label: "Peers", icon: "◌" },
  { path: "/networks", label: "Networks", icon: "⌘" },
  { path: "/profiles", label: "Profiles", icon: "◎" },
  { path: "/diagnostics", label: "日志与诊断", icon: "◫" },
  { path: "/client", label: "客户端管理", icon: "⚙" },
];
</script>
<template>
  <div class="shell">
    <FnSidebar :items="links" />
    <main><RouterView /></main>
  </div>
</template>
<style>
@import "./styles/fn.css";
.shell {
  display: flex;
  min-height: 100vh;
}
main {
  flex: 1;
  margin: 20px;
  padding: 26px;
  background: #fff;
  border: 1px solid var(--fn-border);
  border-radius: 16px;
  box-shadow: 0 2px 10px #10182808;
  min-width: 0;
}
</style>
