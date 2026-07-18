<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, json } from '../api'
const profiles=ref<any[]>([]);const name=ref('');const error=ref('')
async function load(){try{profiles.value=await api('/api/profiles')}catch{error.value='Profiles 不可用。'}}
async function add(){try{await api('/api/profiles',json({name:name.value}));name.value='';await load()}catch{error.value='无法新建 Profile。'}}
async function choose(p:any){try{await api(`/api/profiles/${encodeURIComponent(p.id)}/select`,json({}));await load()}catch{error.value='无法切换 Profile。'}}
async function remove(p:any){if(!confirm(`删除 ${p.name}？`))return;try{await api(`/api/profiles/${encodeURIComponent(p.id)}`,{method:'DELETE'});await load()}catch{error.value='无法删除活动或默认 Profile。'}}
onMounted(load)
</script>
<template><section><h2>Profiles</h2><p v-if="error">{{error}}</p><form @submit.prevent="add"><input v-model="name" maxlength="128" placeholder="新 Profile 名称" required><button>新建</button></form><table><thead><tr><th>名称</th><th>ID</th><th>状态</th><th>操作</th></tr></thead><tbody><tr v-for="p in profiles" :key="p.id"><td>{{p.name}}</td><td>{{p.id}}</td><td>{{p.active?'活动':'—'}}</td><td><button @click="choose(p)">切换</button><button @click="remove(p)">删除</button></td></tr></tbody></table></section></template><style scoped>form{margin-bottom:1rem}input{margin-right:.5rem}table{width:100%;border-collapse:collapse}td,th{padding:.5rem;border-bottom:1px solid #ddd;text-align:left}</style>
