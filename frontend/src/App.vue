<script setup lang="ts">
const links = [['/', 'Dashboard'], ['/settings', 'Settings'], ['/logs', 'Logs'], ['/about', 'About']]
</script>

<template>
  <main><header><h1>NetBird for fnOS</h1><nav><RouterLink v-for="[path, name] in links" :key="path" :to="path">{{ name }}</RouterLink></nav></header><RouterView /></main>
</template>

<style>
body { margin: 0; font-family: system-ui, sans-serif; background: #f6f8fa; color: #1f2937; } main { max-width: 960px; margin: auto; padding: 2rem; } header { display: flex; justify-content: space-between; align-items: center; gap: 1rem; } nav { display: flex; gap: 1rem; } a { color: #0969da; } section { background: white; border-radius: 8px; padding: 1.5rem; margin-top: 2rem; } pre { background: #f6f8fa; padding: 1rem; overflow: auto; }
</style>
