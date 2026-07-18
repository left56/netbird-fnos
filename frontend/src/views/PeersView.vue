<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'
const info=ref<any>();const error=ref('')
async function load(){try{info.value=await api('/api/diagnostics')}catch{error.value='无法读取 Peer 连接状态。'}}
onMounted(load)
</script>
<template><section><h2>Peers</h2><p>当前本机与已连接 Peer 的状态由官方 CLI 提供。</p><p v-if="error">{{error}}</p><template v-else><dl v-if="info?.status"><dt>本机状态</dt><dd>{{info.status.state}}</dd><dt>已连接</dt><dd>{{info.status.connected?'是':'否'}}</dd></dl><table v-if="info?.peers?.length"><thead><tr><th>Peer</th><th>IP</th><th>状态</th><th>链路</th></tr></thead><tbody><tr v-for="peer in info.peers" :key="peer.fqdn || peer.ip"><td>{{peer.fqdn || '—'}}</td><td>{{peer.ip || '—'}}</td><td>{{peer.connectionStatus || peer.status || '—'}}</td><td>{{peer.connectionType || '—'}}</td></tr></tbody></table><p v-else>当前没有可显示的 Peer。</p><button @click="load">刷新</button></template></section></template><style scoped>dl{display:grid;grid-template-columns:8rem 1fr;gap:.5rem}dt{font-weight:bold}table{width:100%;border-collapse:collapse;margin:1rem 0}td,th{padding:.5rem;border-bottom:1px solid #ddd;text-align:left}</style>
