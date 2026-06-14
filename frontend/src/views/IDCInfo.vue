<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getIDCInfo, downloadExport } from '../api'

const loading = ref(false)
const list = ref<any[]>([])
const searchText = ref('')

const filtered = computed(() => {
  if (!searchText.value.trim()) return list.value
  const kw = searchText.value.trim().toLowerCase()
  return list.value.filter((item: any) =>
    item.zbx_id?.toLowerCase().includes(kw) ||
    item.ipmi_ip?.toLowerCase().includes(kw) ||
    item.ssh_ip?.toLowerCase().includes(kw)
  )
})

onMounted(fetchData)

function doExport() { downloadExport('/idc_info/export', {}) }

async function fetchData() {
  loading.value = true
  try {
    const res = await getIDCInfo()
    if (res.code === 200) {
      list.value = res.data || []
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="page-header">
    <h2>SSH 信息 (IDC 机房关联)</h2>
  </div>

  <div class="search-bar">
    <div class="search-field">
      <label>搜索</label>
      <input v-model="searchText" placeholder="按 ZbxID / IPMI / SSH 过滤" />
    </div>
    <button @click="fetchData">刷新</button>
    <button style="background:#52c41a;color:#fff;border:none" @click="doExport">导出Excel</button>
  </div>

  <div class="data-card">
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="filtered.length === 0" class="empty">暂无数据</div>
    <template v-else>
      <div class="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>ZbxID</th>
              <th>IPMI IP</th>
              <th>SSH IP</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, idx) in filtered" :key="idx">
              <td>{{ item.zbx_id }}</td>
              <td>{{ item.ipmi_ip }}</td>
              <td>{{ item.ssh_ip }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="table-footer">
        <span class="total">共 {{ filtered.length }} 条记录</span>
      </div>
    </template>
  </div>
</template>
