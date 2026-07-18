<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, json } from '../api'
const networks=ref<any[]>([]);const error=ref('')
async function load(){try{networks.value=await api('/api/networks')}catch{error.value='Networks 不可用。'}}
async function toggle(n:any){try{await api(n.selected?'/api/networks/deselect':'/api/networks/select',json({ids:[n.id]}));await load()}catch{error.value='网络选择失败。'}}
onMounted(load)
</script>
<template><section><h2>Networks</h2><p>重叠网络由 NetBird 服务端策略处理；此页面只调用官方选择接口，不自行修改路由。</p><p v-if="error">{{error}}</p><table><thead><tr><th>网络</th><th>ID</th><th>选择</th><th>Exit Node</th><th>操作</th></tr></thead><tbody><tr v-for="n in networks" :key="n.id"><td>{{n.name}}</td><td>{{n.id}}</td><td>{{n.selected?'已选择':'未选择'}}</td><td>{{n.exitNode?'是':'—'}}</td><td><button @click="toggle(n)">{{n.selected?'取消':'选择'}}</button></td></tr></tbody></table></section></template><style scoped>table{width:100%;border-collapse:collapse}td,th{padding:.5rem;border-bottom:1px solid #ddd;text-align:left}</style>
