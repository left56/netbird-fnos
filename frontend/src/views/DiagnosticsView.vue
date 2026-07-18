<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
const data=ref<any>();const error=ref('')
async function load(){try{data.value=await api('/api/diagnostics')}catch{error.value='诊断不可用。'}}
onMounted(load)
</script>
<template><section><h2>日志与诊断</h2><p>只显示官方 CLI 的安全状态摘要；认证令牌和密钥不会返回。</p><p v-if="error">{{error}}</p><dl v-else-if="data?.status"><dt>连接状态</dt><dd>{{data.status.state}}</dd><dt>连接</dt><dd>{{data.status.connected?'是':'否'}}</dd></dl><button @click="load">刷新诊断</button></section></template><style scoped>dl{display:grid;grid-template-columns:8rem 1fr;gap:.5rem}dt{font-weight:bold}</style>
