<script setup lang="ts">
import { ref } from 'vue'
import { api, json } from '../api'
const settings=ref({allowServerSSH:false,blockInbound:false,blockLANAccess:false,disableAutoConnect:false,disableClientRoutes:false});const message=ref('')
async function connect(){try{await api('/api/connect',json(settings.value));message.value='连接请求已发送。'}catch{message.value='设置或连接失败。'}}
</script>
<template><section><h2>设置</h2><p>常用动态参数将在下一次 Connect 时传递给官方 CLI。</p><label v-for="(value,key) in settings" :key="key"><input v-model="settings[key]" type="checkbox"> {{key}}</label><p><button @click="connect">应用并连接</button></p><p>{{message}}</p></section></template><style scoped>label{display:block;margin:.6rem 0}</style>
