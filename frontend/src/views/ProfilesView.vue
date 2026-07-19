<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api, json } from "../api";
import FnButton from "../components/FnButton.vue";
import FnCard from "../components/FnCard.vue";
import FnDialog from "../components/FnDialog.vue";
import FnFormItem from "../components/FnFormItem.vue";
import FnInput from "../components/FnInput.vue";
import FnPageHeader from "../components/FnPageHeader.vue";
import FnSwitch from "../components/FnSwitch.vue";
import FnTag from "../components/FnTag.vue";
const profiles = ref<any[]>([]),
  error = ref(""),
  createOpen = ref(false),
  editOpen = ref(false),
  editingID = ref(""),
  authMessage = ref(""),
  draft = ref({
    name: "",
    managementURL: "",
    setupKey: "",
    presharedKey: "",
    selectAfterCreate: true,
    connectAfterCreate: false,
  });
const editDraft = ref<any>({});
async function load() {
  try {
    profiles.value = (await api<any[]>("/api/profiles")).map((detail) => ({ ...detail.metadata, ...detail.runtime, config: detail.config, source: detail.source }));
    error.value = "";
  } catch (e) {
    error.value = "当前版本不支持多 Profile，或官方客户端不可用。";
  }
}
async function create() {
  try {
    await api("/api/profiles", json({
      name: draft.value.name,
      config: { managementURL: draft.value.managementURL },
      setupKey: draft.value.setupKey,
      presharedKey: draft.value.presharedKey,
      selectAfterCreate: draft.value.selectAfterCreate,
      connectAfterCreate: draft.value.setupKey !== "" || draft.value.connectAfterCreate,
    }));
    createOpen.value = false;
    await load();
  } catch {
    error.value = "无法创建 Profile。";
  }
}
async function select(p: any) {
  if (p.active || !confirm("切换 Profile 会中断当前连接，继续吗？")) return;
  await api(`/api/profiles/${encodeURIComponent(p.id)}/select`, json({}));
  await load();
}
async function remove(p: any) {
  if (p.default || p.active) {
    error.value = "default 或当前 Profile 不能删除。";
    return;
  }
  if (!confirm(`删除 ${p.name}？`)) return;
  try {
    await api(`/api/profiles/${encodeURIComponent(p.id)}`, {
      method: "DELETE",
    });
    await load();
  } catch {
    error.value = "已连接的 Profile 必须先断开。";
  }
}
function edit(p: any) {
  editingID.value = p.id;
  editDraft.value = { ...p.config, name: p.config?.name || p.name || "", managementURL: p.config?.managementURL || "" };
  authMessage.value = "";
  editOpen.value = true;
}
async function saveEdit() {
  try {
    await api(`/api/profiles/${encodeURIComponent(editingID.value)}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ config: editDraft.value }) });
    editOpen.value = false;
    await load();
  } catch {
    error.value = "无法保存 Profile 配置。";
  }
}
async function authenticateWithSetupKey() {
  const setupKey = prompt("输入一次性或可复用的 NetBird Setup Key（不会保存或回显）：");
  if (!setupKey) return;
  try {
    await api("/api/connect", json({ managementURL: editDraft.value.managementURL, setupKey }));
    authMessage.value = "认证请求已提交，正在连接。";
    await load();
  } catch {
    authMessage.value = "认证或连接失败；请检查 Setup Key 与 Management URL。";
  }
}
onMounted(load);
</script>
<template>
  <FnPageHeader
    title="Profiles"
    description="在不同的 NetBird 帐户与网络配置之间安全切换"
    ><template #default
      ><FnButton variant="primary" @click="createOpen = true"
        >新建 Profile</FnButton
      ></template
    ></FnPageHeader
  >
  <p v-if="error" class="error">{{ error }}</p>
  <div class="grid">
    <FnCard v-for="p in profiles" :key="p.id" class="profile"
      ><div class="card-head">
        <div>
          <h3>{{ p.name }}</h3>
          <small>{{ p.id }}</small>
        </div>
        <div>
          <FnTag v-if="p.active" type="success">当前活动</FnTag
          ><FnTag v-if="p.default" type="primary">默认</FnTag>
        </div>
      </div>
      <dl>
        <dt>连接状态</dt>
        <dd>{{ p.connected ? "已连接" : "未连接" }}</dd>
        <dt>Management URL</dt>
        <dd>{{ p.config?.managementURL || "未配置" }}</dd>
        <dt>NetBird IP</dt>
        <dd>{{ p.netbirdIP || "—" }}</dd>
        <dt>Networks / Exit Node</dt>
        <dd>{{ p.enabledNetworks || 0 }} 个 / {{ p.exitNode || "未选择" }}</dd>
        <dt>最后连接</dt>
        <dd>{{ p.lastConnectedAt || "—" }}</dd>
      </dl>
      <div class="actions">
        <FnButton @click="select(p)">切换</FnButton><FnButton @click="edit(p)">编辑配置</FnButton
        ><FnButton
          :disabled="p.default || p.active"
          variant="danger"
          @click="remove(p)"
          >删除</FnButton
        >
      </div></FnCard
    >
  </div>
  <FnDialog :open="createOpen" title="新建 Profile" @close="createOpen = false"
    ><div class="form">
      <FnFormItem label="Profile 名称"
        ><FnInput
          v-model="draft.name"
          placeholder="例如：工作网络" /></FnFormItem
      ><FnFormItem label="Management URL"
        ><FnInput
          v-model="draft.managementURL"
          placeholder="https://api.netbird.io" /></FnFormItem
      ><FnFormItem label="Setup Key（可选）"
        ><FnInput v-model="draft.setupKey" type="password" /></FnFormItem
      ><FnFormItem label="Pre-shared Key（可选）"
        ><FnInput v-model="draft.presharedKey" type="password" /></FnFormItem
      ><label>创建后切换 <FnSwitch v-model="draft.selectAfterCreate" /></label
      ><label>创建后连接 <FnSwitch v-model="draft.connectAfterCreate" /></label>
    </div>
    <template #footer
      ><FnButton @click="createOpen = false">取消</FnButton
      ><FnButton variant="primary" @click="create">创建</FnButton></template
    ></FnDialog
  >
  <FnDialog :open="editOpen" title="编辑 Profile 配置" @close="editOpen = false">
    <div class="form">
      <FnFormItem label="Profile 名称"><FnInput v-model="editDraft.name" /></FnFormItem>
      <FnFormItem label="Management URL"><FnInput v-model="editDraft.managementURL" placeholder="https://api.netbird.io" /></FnFormItem>
      <p class="hint">NAS 无桌面环境建议使用 Setup Key 认证；密钥仅随本次请求传给官方 NetBird CLI，不会被保存或显示。</p>
      <p v-if="authMessage" :class="authMessage.startsWith('认证或') ? 'error' : 'hint'">{{ authMessage }}</p>
    </div>
    <template #footer>
      <FnButton @click="authenticateWithSetupKey">使用 Setup Key 认证</FnButton>
      <FnButton @click="editOpen = false">取消</FnButton>
      <FnButton variant="primary" @click="saveEdit">保存</FnButton>
    </template>
  </FnDialog>
</template>
<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 16px;
}
.profile {
  padding: 18px;
}
.card-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}
.card-head h3 {
  margin: 0;
  font-size: 16px;
}
.card-head small {
  color: var(--fn-muted);
}
dl {
  display: grid;
  grid-template-columns: 125px 1fr;
  gap: 8px;
  margin: 18px 0;
  font-size: 13px;
}
dt {
  color: var(--fn-muted);
}
dd {
  margin: 0;
  overflow-wrap: anywhere;
}
.actions {
  display: flex;
  gap: 8px;
}
.form {
  display: grid;
  gap: 15px;
}
.form label {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.error {
  color: #c43226;
}
.hint {
  margin: 0;
  color: var(--fn-muted);
  font-size: 13px;
  line-height: 1.5;
}
</style>
