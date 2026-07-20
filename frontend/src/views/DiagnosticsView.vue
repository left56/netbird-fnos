<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api } from "../api"; import FnButton from "../components/FnButton.vue"; import FnCard from "../components/FnCard.vue"; import FnPageHeader from "../components/FnPageHeader.vue";
const lines=ref<string[]>([]), error=ref("");
async function load(){try{const result=await api<any>("/api/logs/latest");lines.value=result.lines||[];error.value=""}catch{error.value="日志暂不可用。"}}
async function copy(){try{await navigator.clipboard.writeText(lines.value.join("\n"))}catch{error.value="无法复制日志。"}}
function download(){const blob=new Blob([lines.value.join("\n")+"\n"],{type:"text/plain"}),url=URL.createObjectURL(blob),a=document.createElement("a");a.href=url;a.download="netbird-fnos-latest.log";a.click();URL.revokeObjectURL(url)}
onMounted(load);
</script>
<template><FnPageHeader title="日志与诊断" description="最近 100 行包装器日志；潜在敏感字段会被隐藏"><template #default><FnButton @click="load">刷新</FnButton><FnButton @click="copy">复制</FnButton><FnButton variant="primary" @click="download">下载</FnButton></template></FnPageHeader><FnCard><p v-if="error" class="error">{{error}}</p><pre v-else>{{lines.length ? lines.join('\n') : '暂无日志。'}}</pre></FnCard></template>
<style scoped>pre{margin:0;max-height:620px;overflow:auto;padding:14px;border-radius:10px;background:#111827;color:#d1d5db;font:12px/1.6 ui-monospace,SFMono-Regular,Menlo,monospace;white-space:pre-wrap;word-break:break-word}.error{color:#c43226}</style>
