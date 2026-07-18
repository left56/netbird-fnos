<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiURL } from '../gateway'

const client = ref<any>()
const versions = ref<any[]>([])
const message = ref('')
async function request(path: string, init?: RequestInit) {
  const response = await fetch(apiURL(path), init)
  const body = await response.json()
  if (!response.ok) throw new Error(body.data?.message || 'Request failed')
  return body.data
}
async function refresh() { try { [client.value, versions.value] = await Promise.all([request('/api/client'), request('/api/client/versions')]) } catch { message.value = '无法读取客户端状态。' } }
async function action(path: string, body?: object) { if (!confirm('确认执行此危险操作？')) return; try { await request(path, { method: 'POST', headers: {'Content-Type':'application/json'}, body: body ? JSON.stringify(body) : undefined }); message.value='操作完成。'; await refresh() } catch (error) { message.value = error instanceof Error ? error.message : '操作失败。' } }
async function removeVersion(version: string) { if (!confirm(`删除历史版本 ${version}？`)) return; try { await request(`/api/client/versions/${encodeURIComponent(version)}`, {method:'DELETE'}); await refresh() } catch { message.value='无法删除当前或受保护版本。' } }
onMounted(() => void refresh())
</script>
<template>
  <section><h2>客户端管理</h2><p>管理内置与在线安装的官方 NetBird CLI。绝对路径不会暴露或可编辑。</p><p v-if="message">{{ message }}</p>
    <template v-if="client"><dl><dt>当前版本</dt><dd>{{ client.version || '不可用' }}</dd><dt>来源</dt><dd>{{ client.source }}</dd><dt>架构</dt><dd>{{ client.arch || '—' }}</dd><dt>Checksum</dt><dd>{{ client.checksum || '—' }}</dd><dt>能力</dt><dd><code>{{ JSON.stringify(client.capabilities) }}</code></dd></dl></template>
    <p><button @click="refresh">检查更新</button> <button @click="action('/api/client/use-bundled')">回退内置版本</button></p>
    <h3>已安装版本</h3><table><thead><tr><th>版本</th><th>架构</th><th>操作</th></tr></thead><tbody><tr v-for="item in versions" :key="item.version"><td>{{item.version}}</td><td>{{item.arch}}</td><td><button @click="action('/api/client/switch',{version:item.version})">切换</button> <button @click="removeVersion(item.version)">删除</button></td></tr></tbody></table>
    <h3>在线安装、上传与下载源</h3><p>该构建保留受控 API 与管理员权限边界；安装源、上传暂未启用，直到发布解析器提供官方 checksum。TLS 校验固定开启，第三方加速器仅可作为传输层。</p>
  </section>
</template>
<style scoped>dl{display:grid;grid-template-columns:10rem 1fr;gap:.5rem}dt{font-weight:600}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid #ddd;padding:.5rem;text-align:left}button{margin-right:.4rem}</style>
