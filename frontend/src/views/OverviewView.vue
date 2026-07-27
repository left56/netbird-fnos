<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { api, json } from "../api";
import FnButton from "../components/FnButton.vue";
import FnCard from "../components/FnCard.vue";
import FnPageHeader from "../components/FnPageHeader.vue";
import FnTag from "../components/FnTag.vue";

const status = ref<any>();
const peers = ref<any[]>([]);
const error = ref("");
let timer: number | undefined;
const connected = computed(() => !!status.value?.connection?.connected);
const recentPeers = computed(() => peers.value.filter((peer) => peer.connected).slice(0, 6));
async function refresh() {
  try {
    const [nextStatus, nextPeers] = await Promise.all([api<any>("/api/status"), api<any[]>("/api/peers")]);
    status.value = nextStatus; peers.value = nextPeers; error.value = "";
  } catch { error.value = "无法读取 NetBird 运行状态。"; }
}
async function toggle() {
  try { await api(connected.value ? "/api/disconnect" : "/api/connect", json({})); await refresh(); }
  catch { error.value = connected.value ? "断开连接失败。" : "连接失败。"; }
}
onMounted(() => { void refresh(); timer = window.setInterval(refresh, 15000); });
onBeforeUnmount(() => { if (timer) window.clearInterval(timer); });
</script>
<template>
  <FnPageHeader title="概览" description="连接状态与对等节点">
    <template #default><FnButton @click="refresh">刷新</FnButton><FnButton variant="primary" @click="toggle">{{ connected ? "断开连接" : "连接" }}</FnButton></template>
  </FnPageHeader>
  <p v-if="error" class="error">{{ error }}</p>
  <div class="workspace">
    <FnCard class="identity">
      <div class="profile-line"><span class="avatar">◎</span><div><small>当前 Profile</small><h2>{{ status?.profile?.name || "default" }}</h2></div><FnTag :type="connected ? 'success' : 'muted'">{{ connected ? "已连接" : "未连接" }}</FnTag></div>
      <div class="connection-mark" :class="{ connected }"><span>◆</span></div>
      <h3>{{ connected ? "已连接" : "等待连接" }}</h3>
      <p class="account">{{ status?.device?.hostname || "NetBird peer" }}</p>
      <code>{{ status?.device?.netbirdIp || "尚未分配 NetBird IP" }}</code>
      <div class="exit"><span>⇄</span><div><strong>出口节点</strong><small> {{ status?.exitNode || "未启用" }}</small></div><RouterLink to="/networks">管理</RouterLink></div>
    </FnCard>
    <FnCard class="peer-panel">
      <header><div><h2>对等节点</h2><p>在线 {{ status?.statistics?.onlinePeers ?? 0 }} / {{ status?.statistics?.peerCount ?? 0 }} · Direct {{ status?.statistics?.directPeers ?? 0 }}</p></div><RouterLink to="/peers">查看全部</RouterLink></header>
      <div v-if="recentPeers.length" class="peer-list"><div v-for="peer in recentPeers" :key="peer.id" class="peer"><span class="dot" :class="{ relay: !peer.direct }"></span><div><strong>{{ peer.name || peer.id }}</strong><small>{{ peer.ip || "—" }}</small></div><em v-if="peer.latencyMs">{{ peer.latencyMs }} ms</em><FnTag :type="peer.direct ? 'success' : 'warning'">{{ peer.direct ? "Direct" : "Relay" }}</FnTag></div></div>
      <div v-else class="empty">暂无在线 Peer</div>
      <footer><div><small>Management</small><strong>{{ status?.connection?.management || "—" }}</strong></div><div><small>Signal</small><strong>{{ status?.connection?.signal || "—" }}</strong></div><div><small>已启用 Networks</small><strong>{{ status?.statistics?.enabledNetworks ?? 0 }}</strong></div></footer>
    </FnCard>
  </div>
  <div class="summary"><FnCard><small>NetBird 版本</small><strong>{{ status?.versions?.netbird || "—" }}</strong><span>{{ status?.versions?.source || "bundled" }}</span></FnCard><FnCard><small>接口</small><strong>{{ status?.device?.interface || "wt0" }}</strong><span>{{ status?.device?.publicKey ? "密钥已就绪" : "等待 daemon" }}</span></FnCard><FnCard><small>连接方式</small><strong>{{ status?.statistics?.relayPeers ?? 0 ? "含 Relay" : "Direct" }}</strong><span>Relay {{ status?.statistics?.relayPeers ?? 0 }} 个</span></FnCard></div>
</template>
<style scoped>
.workspace{display:grid;grid-template-columns:minmax(280px,.8fr) minmax(420px,1.3fr);gap:16px}.identity{text-align:center;min-height:480px;display:flex;flex-direction:column}.profile-line{display:flex;align-items:center;text-align:left;gap:10px}.profile-line div{flex:1}.profile-line small,.peer small,small{color:var(--fn-muted)}.profile-line h2{margin:2px 0 0;font-size:17px}.avatar{display:grid;place-items:center;width:34px;height:34px;border-radius:50%;background:var(--fn-primary-soft);color:var(--fn-primary);font-size:20px}.connection-mark{margin:58px auto 16px;display:grid;place-items:center;width:78px;height:48px;border-radius:28px;background:#dce2ea;color:#fff}.connection-mark span{font-size:22px}.connection-mark.connected{background:#2bbf88}.identity h3{margin:4px 0;font-size:21px}.account{margin:0;color:var(--fn-muted)}code{display:block;margin:9px 0;color:#526072;font-size:15px}.exit{display:flex;align-items:center;gap:10px;text-align:left;margin-top:auto;padding:13px;border:1px solid var(--fn-border);border-radius:12px;background:#fafbfd}.exit>span{font-size:22px;color:var(--fn-primary)}.exit div{display:grid;gap:2px;flex:1}.exit a,.peer-panel header>a{color:var(--fn-primary);text-decoration:none;font-size:13px;font-weight:600}.peer-panel{padding:0}.peer-panel header{display:flex;justify-content:space-between;align-items:start;padding:20px;border-bottom:1px solid var(--fn-border)}.peer-panel h2{margin:0;font-size:18px}.peer-panel header p{margin:5px 0 0;color:var(--fn-muted);font-size:13px}.peer-list{padding:4px 20px}.peer{display:flex;align-items:center;gap:12px;padding:13px 0;border-bottom:1px solid var(--fn-border)}.peer>div{display:grid;gap:3px;flex:1}.peer em{font-style:normal;color:#9a6700;font-size:13px}.dot{width:10px;height:10px;border-radius:50%;background:#21b983}.dot.relay{background:#e4a11b}.peer-panel footer{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;padding:18px 20px;background:#fafbfd}.peer-panel footer div{display:grid;gap:4px}.peer-panel footer strong{font-size:13px}.empty{display:grid;place-items:center;min-height:210px;color:var(--fn-muted)}.summary{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin-top:16px}.summary section{display:grid;gap:7px}.summary strong{font-size:18px}.summary span{font-size:13px;color:var(--fn-muted)}.error{color:#c43226}@media(max-width:860px){.workspace{grid-template-columns:1fr}.summary{grid-template-columns:1fr}}
</style>
