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
  draft = ref({
    name: "",
    managementURL: "",
    setupKey: "",
    presharedKey: "",
    selectAfterCreate: true,
    connectAfterCreate: false,
  });
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
      connectAfterCreate: draft.value.connectAfterCreate,
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
        <dd>{{ p.managementURL || "未配置" }}</dd>
        <dt>NetBird IP</dt>
        <dd>{{ p.netbirdIP || "—" }}</dd>
        <dt>Networks / Exit Node</dt>
        <dd>{{ p.enabledNetworks || 0 }} 个 / {{ p.exitNode || "未选择" }}</dd>
        <dt>最后连接</dt>
        <dd>{{ p.lastConnectedAt || "—" }}</dd>
      </dl>
      <div class="actions">
        <FnButton @click="select(p)">切换</FnButton><FnButton>编辑配置</FnButton
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
</style>
