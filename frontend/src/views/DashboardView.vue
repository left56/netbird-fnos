<script setup lang="ts">
import { onMounted, ref } from 'vue'

type ApiResponse = { status: string; data: unknown }
const results = ref<Record<string, ApiResponse | { error: string }>>({})

async function load(name: string, path: string) {
  try { results.value[name] = await (await fetch(path)).json() as ApiResponse }
  catch { results.value[name] = { error: 'Local API is unavailable.' } }
}
onMounted(() => { void load('health', '/api/health'); void load('version', '/api/version'); void load('status', '/api/status') })
</script>

<template>
  <section><h2>Dashboard</h2><p>P0 local wrapper status. This is not a production-ready management interface.</p><div v-for="(result, name) in results" :key="name"><h3>{{ name }}</h3><pre>{{ JSON.stringify(result, null, 2) }}</pre></div></section>
</template>
