<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api, json } from "../api";
import FnButton from "../components/FnButton.vue";
import FnCard from "../components/FnCard.vue";
import FnEmpty from "../components/FnEmpty.vue";
import FnPageHeader from "../components/FnPageHeader.vue";
import FnSwitch from "../components/FnSwitch.vue";
import FnTabs from "../components/FnTabs.vue";
import FnTag from "../components/FnTag.vue";
const all = ref<any[]>([]),
  selected = ref<Set<string>>(new Set()),
  tab = ref("all"),
  error = ref("");
const dirty = computed(() =>
  all.value.some((n) => n.selected !== selected.value.has(n.id)),
);
const shown = computed(() =>
  tab.value === "exit"
    ? all.value.filter((n) => n.exitNode)
    : tab.value === "overlap"
      ? all.value.filter((n) => n.overlap)
      : all.value,
);
async function load() {
  try {
    const result = await api<any>("/api/networks");
    all.value = result.all || [];
    selected.value = new Set(
      all.value.filter((n) => n.selected).map((n) => n.id),
    );
  } catch {
    error.value = "Networks 不可用。";
  }
}
function selectShown(on: boolean) { const next = new Set(selected.value); shown.value.forEach((network) => on ? next.add(network.id) : next.delete(network.id)); selected.value = next; }
function toggle(id: string, on: boolean) {
  const next = new Set(selected.value);
  on ? next.add(id) : next.delete(id);
  selected.value = next;
}
async function apply() {
  const before = new Set(all.value.filter((n) => n.selected).map((n) => n.id)),
    after = selected.value;
  const add = [...after].filter((id) => !before.has(id)),
    remove = [...before].filter((id) => !after.has(id));
  if (
    !confirm(
      `将启用 ${add.length} 个、停用 ${remove.length} 个 Network，继续吗？`,
    )
  )
    return;
  try {
    if (add.length) await api("/api/networks/select", json({ ids: add }));
    if (remove.length)
      await api("/api/networks/deselect", json({ ids: remove }));
    await load();
  } catch {
    error.value = "应用 Network 更改失败。";
  }
}
onMounted(load);
</script>
<template>
  <FnPageHeader
    title="Networks"
    description="先暂存选择，再统一应用到官方 NetBird CLI"
  ><template #default><FnButton @click="selectShown(true)">全选当前</FnButton><FnButton @click="selectShown(false)">取消当前</FnButton></template></FnPageHeader><FnCard
    ><FnTabs v-model="tab"
      ><template #default="{ select, active }"
        ><button :class="{ active: active === 'all' }" @click="select('all')">
          全部 Networks</button
        ><button
          :class="{ active: active === 'overlap' }"
          @click="select('overlap')"
        >
          重叠 Networks</button
        ><button :class="{ active: active === 'exit' }" @click="select('exit')">
          Exit Node
        </button></template
      ></FnTabs
    >
    <p v-if="error" class="error">{{ error }}</p>
    <div v-if="shown.length" class="list">
      <div v-for="network in shown" :key="network.id" class="row">
        <FnSwitch
          :model-value="selected.has(network.id)"
          @update:model-value="toggle(network.id, $event)"
        />
        <div>
          <strong>{{ network.name }}</strong
          ><small>{{ network.id }}</small>
        </div>
        <FnTag v-if="network.exitNode" type="primary">Exit Node</FnTag
        ><FnTag v-if="network.overlap" type="warning">重叠</FnTag>
      </div>
    </div>
    <FnEmpty v-else>此分类没有 Networks</FnEmpty></FnCard
  >
  <footer v-if="dirty" class="bar">
    <span>存在未保存的 Network 更改</span>
    <div>
      <FnButton @click="load">取消更改</FnButton
      ><FnButton variant="primary" @click="apply">应用更改</FnButton>
    </div>
  </footer>
</template>
<style scoped>
button {
  border: 0;
  background: transparent;
  padding: 0 0 10px;
  color: var(--fn-muted);
}
button.active {
  color: var(--fn-primary);
  border-bottom: 2px solid var(--fn-primary);
}
.list {
  display: grid;
}
.row {
  display: flex;
  align-items: center;
  gap: 13px;
  padding: 14px 4px;
  border-bottom: 1px solid var(--fn-border);
}
.row div {
  display: grid;
  gap: 3px;
  flex: 1;
}
.row small {
  color: var(--fn-muted);
}
.bar {
  position: sticky;
  bottom: 0;
  margin-top: 16px;
  padding: 14px 18px;
  background: #fff;
  border: 1px solid var(--fn-border);
  border-radius: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.bar div {
  display: flex;
  gap: 8px;
}
.error {
  color: #c43226;
}
</style>
