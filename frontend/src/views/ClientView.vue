<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, json } from '../api'

const client = ref<any>(); const sources = ref<any[]>([])
const versions = ref<any[]>([])
const message = ref(''); const version = ref(''); const sourceID = ref(''); const allowUnverified = ref(false); const file = ref<HTMLInputElement>(); const source = ref({name:'',enabled:true,priority:10,urlTemplate:'',timeoutSeconds:60})
async function refresh() { try { [client.value, versions.value, sources.value] = await Promise.all([api<any>('/api/client'), api<any[]>('/api/client/versions'), api<any[]>('/api/download-sources')]) } catch { message.value = '无法读取客户端状态。' } }
async function action(path: string, body?: object) { if (!confirm('确认执行此危险操作？')) return; try { await api(path, json(body || {})); message.value='操作完成。'; await refresh() } catch (error) { message.value = error instanceof Error ? error.message : '操作失败。' } }
async function removeVersion(v: string) { if (!confirm(`删除历史版本 ${v}？`)) return; try { await api(`/api/client/versions/${encodeURIComponent(v)}`, {method:'DELETE'}); await refresh() } catch { message.value='无法删除当前或受保护版本。' } }
async function download(){try{await api('/api/client/download',json({version:version.value,sourceId:sourceID.value}));message.value='已校验并安装，随后可切换版本。';await refresh()}catch{message.value='下载或官方 checksum 校验失败。'}}
async function checkUpdate(){try{const release:any=await api('/api/client/check-update',json({version:''}));version.value=release.version;message.value=`可用官方版本：${release.version}`;}catch{message.value='无法查询官方版本。'}}
async function upload(){const selected=file.value?.files?.[0];if(!selected)return;if(allowUnverified.value&&!confirm('此文件未通过官方来源校验。确认以 upload-unverified 安装？'))return;const body=new FormData();body.append('file',selected);body.append('allowUnverified',String(allowUnverified.value));try{await api('/api/client/upload',{method:'POST',body});message.value='上传已验证并安装。';await refresh()}catch{message.value='上传被拒绝：未校验上传必须明确高级确认。'}}
async function saveSource(){try{await api('/api/download-sources',json(source.value));source.value={name:'',enabled:true,priority:10,urlTemplate:'',timeoutSeconds:60};await refresh()}catch{message.value='下载源必须使用 HTTPS 且 URL 模板只包含一个 {url}。'}}
async function testSource(id:string){try{await api(`/api/download-sources/${encodeURIComponent(id)}/test`,json({}));message.value='下载源连通。'}catch{message.value='下载源测试失败。'}}
async function removeSource(id:string){if(!confirm('删除下载源？'))return;await api(`/api/download-sources/${encodeURIComponent(id)}`,{method:'DELETE'});await refresh()}
async function moveSource(s:any, delta:number){try{await api(`/api/download-sources/${encodeURIComponent(s.id)}`,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({...s,priority:Math.max(0,s.priority+delta)})});await refresh()}catch{message.value='无法调整下载源排序。'}}
async function editSource(s:any){const urlTemplate=prompt('HTTPS URL 模板（必须含 {url}）',s.urlTemplate);if(!urlTemplate)return;const name=prompt('名称',s.name);if(!name)return;try{await api(`/api/download-sources/${encodeURIComponent(s.id)}`,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({...s,name,urlTemplate})});await refresh()}catch{message.value='下载源配置无效。'}}
onMounted(() => void refresh())
</script>
<template>
  <section><h2>客户端管理</h2><p>管理内置与在线安装的官方 NetBird CLI。绝对路径不会暴露或可编辑。</p><p v-if="message">{{ message }}</p>
    <template v-if="client"><dl><dt>当前版本</dt><dd>{{ client.version || '不可用' }}</dd><dt>来源</dt><dd>{{ client.source }}</dd><dt>架构</dt><dd>{{ client.arch || '—' }}</dd><dt>Checksum</dt><dd>{{ client.checksum || '—' }}</dd><dt>能力</dt><dd><code>{{ JSON.stringify(client.capabilities) }}</code></dd></dl></template>
    <p><button @click="refresh">刷新</button> <button @click="action('/api/client/use-bundled')">回退内置版本</button> <button @click="action('/api/client/rollback')">回滚上一版本</button></p>
    <h3>已安装版本</h3><table><thead><tr><th>版本</th><th>架构</th><th>操作</th></tr></thead><tbody><tr v-for="item in versions" :key="item.version"><td>{{item.version}}</td><td>{{item.arch}}</td><td><button @click="action('/api/client/switch',{version:item.version})">切换</button> <button @click="removeVersion(item.version)">删除</button></td></tr></tbody></table>
    <h3>在线安装</h3><p>只接受指定官方 Release 版本；最终依据官方 checksum 校验。</p><input v-model="version" placeholder="例如 0.71.4"><select v-model="sourceID"><option value="">官方源</option><option v-for="s in sources.filter(s=>s.enabled)" :key="s.id" :value="s.id">{{s.name}}</option></select><button @click="checkUpdate">检查更新</button><button @click="download">下载并安装</button>
    <h3>手动上传</h3><p>接受官方 ELF 或官方 tar.gz。未能由官方 checksum 验证的上传需要高级确认，默认不会激活。</p><input ref="file" type="file" accept="application/gzip,.gz,application/octet-stream"><label><input v-model="allowUnverified" type="checkbox"> 高级确认：允许未通过官方来源校验的上传</label><button @click="upload">验证并安装</button>
    <h3>GitHub 加速源</h3><p>默认没有启用第三方源。TLS 校验固定开启；模板中的 <code>{url}</code> 将替换为完整官方 URL 的 URL-encoded 值。</p><form @submit.prevent="saveSource"><input v-model="source.name" placeholder="名称" required><input v-model="source.urlTemplate" placeholder="https://proxy.example/{url}" required><input v-model.number="source.priority" type="number" min="0"><input v-model.number="source.timeoutSeconds" type="number" min="1" max="300"><button>新增</button></form><table><tbody><tr v-for="s in sources" :key="s.id"><td>{{s.name}}</td><td>{{s.priority}}</td><td>{{s.enabled?'启用':'禁用'}}</td><td><button @click="moveSource(s,-1)">↑</button><button @click="moveSource(s,1)">↓</button><button @click="editSource(s)">编辑</button><button @click="testSource(s.id)">测试</button><button @click="removeSource(s.id)">删除</button></td></tr></tbody></table>
  </section>
</template>
<style scoped>dl{display:grid;grid-template-columns:10rem 1fr;gap:.5rem}dt{font-weight:600}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid #ddd;padding:.5rem;text-align:left}button,input{margin:.25rem}.state{font-weight:bold}</style>
