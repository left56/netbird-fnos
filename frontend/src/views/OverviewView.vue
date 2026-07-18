<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, json } from '../api'
const status = ref<any>(); const error = ref('')
async function refresh(){try{status.value=await api('/api/status')}catch{error.value='无法读取 NetBird 状态。'}}
async function toggle(){try{if(status.value?.connected) await api('/api/disconnect',json({}));else await api('/api/connect',json({}));await refresh()}catch{error.value='操作失败，请检查客户端权限和守护进程。'}}
onMounted(refresh)
</script>
<template><section><h2>概览</h2><p v-if="error">{{error}}</p><template v-else-if="status"><p class="state" :class="{connected:status.connected}">{{status.connected?'已连接':'未连接'}}</p><dl><dt>状态</dt><dd>{{status.state}}</dd><dt>详情</dt><dd>{{status.detail || '—'}}</dd></dl><button @click="toggle">{{status.connected?'Disconnect':'Connect'}}</button><button @click="refresh">刷新</button></template><p v-else>正在读取状态…</p></section></template>
<style scoped>.state{font-size:1.4rem;font-weight:bold}.connected{color:#087f23}dl{display:grid;grid-template-columns:8rem 1fr;gap:.5rem}dt{font-weight:bold}button{margin-right:.5rem}</style>
