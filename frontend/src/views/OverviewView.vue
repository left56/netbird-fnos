<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { api, json } from "../api";
import FnButton from "../components/FnButton.vue";
import FnCard from "../components/FnCard.vue";
import FnPageHeader from "../components/FnPageHeader.vue";
import FnTag from "../components/FnTag.vue";

const status = ref<any>(); const error = ref(""); let timer: number | undefined;
async function refresh() { try { status.value = await api("/api/status"); error.value = "" } catch { error.value = "无法读取 NetBird 概览。" } }
async function toggle() { try { await api(status.value?.connection?.connected ? "/api/disconnect" : "/api/connect", json({})); await refresh() } catch { error.value = "连接操作失败。" } }
onMounted(() => { refresh(); timer = window.setInterval(refresh, 15000) }); onBeforeUnmount(() => { if (timer) window.clearInterval(timer) });
</script>
<template>
  <FnPageHeader title="概览" description="NetBird 连接、Peer 与 Network 运行状态"><template #default><FnButton @click="refresh">刷新</FnButton><FnButton variant="primary" @click="toggle">{{ status?.connection?.connected ? "断开连接" : "连接" }}</FnButton></template></FnPageHeader>
  <p v-if="error" class="error">{{ error }}</p>
  <div class="stats"><FnCard><small>连接状态</small><h2>{{status?.connection?.connected ? "已连接" : "未连接"}}</h2><FnTag :type="status?.connection?.connected ? 'success' : 'muted'">{{status?.connection?.management || "读取中"}}</FnTag></FnCard><FnCard><small>当前 Profile</small><h2>{{status?.profile?.name || "default"}}</h2><p>{{status?.profile?.id || "—"}}</p></FnCard><FnCard><small>Peers</small><h2>{{status?.statistics?.onlinePeers ?? "—"}} / {{status?.statistics?.peerCount ?? "—"}}</h2><p>Direct {{status?.statistics?.directPeers ?? 0}} · Relay {{status?.statistics?.relayPeers ?? 0}}</p></FnCard><FnCard><small>Networks</small><h2>{{status?.statistics?.enabledNetworks ?? "—"}}</h2><p>已启用 Networks</p></FnCard></div>
  <div class="details"><FnCard><h3>NetBird 信息</h3><dl><dt>NetBird IP</dt><dd>{{status?.device?.netbirdIp || "—"}}</dd><dt>Interface</dt><dd>{{status?.device?.interface || "—"}}</dd><dt>Management</dt><dd>{{status?.connection?.management || "—"}}</dd><dt>Signal</dt><dd>{{status?.connection?.signal || "—"}}</dd></dl></FnCard><FnCard><h3>版本</h3><dl><dt>NetBird</dt><dd>{{status?.versions?.netbird || "—"}}</dd><dt>Wrapper</dt><dd>{{status?.versions?.wrapper || "—"}}</dd><dt>来源</dt><dd>{{status?.versions?.source || "—"}}</dd></dl></FnCard></div>
</template>
<style scoped>.stats,.details{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px}.details{margin-top:16px}.stats small,.stats p,dt{color:var(--fn-muted)}.stats h2{margin:12px 0 7px;font-size:23px}.stats p{margin:0;font-size:13px}h3{margin:0 0 14px}dl{display:grid;grid-template-columns:120px 1fr;gap:10px;margin:0;font-size:13px}dd{margin:0}.error{color:#c43226}</style>
