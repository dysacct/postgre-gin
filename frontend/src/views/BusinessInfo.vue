<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listMachines, downloadExport } from '../api'

const loading = ref(false)
const total = ref(0)
const page = ref(1)
const size = ref(300)
const list = ref<any[]>([])
const searchText = ref('')
const idcCode = ref('')
const showUnknown = ref(true)

const totalPages = computed(() => Math.ceil(total.value / size.value) || 1)

const filteredList = computed(() => {
  if (showUnknown.value) return list.value
  return list.value.filter((item: any) =>
    item.business_info?.business_name &&
    item.business_info.business_name !== '未知业务'
  )
})

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, string> = { page: String(page.value), size: String(size.value) }
    if (searchText.value.trim()) {
      params.business_name = searchText.value.trim()
    }
    if (idcCode.value.trim()) {
      params.idc_code = idcCode.value.trim()
    }
    const res = await listMachines(params)
    if (res.code === 200) {
      total.value = res.data?.total || 0
      // 保留所有记录，包括"未知业务"
      list.value = res.data?.list || []
    }
  } finally {
    loading.value = false
  }
}

function isUnknown(item: any): boolean {
  return item.business_info?.business_name === '未知业务' ||
         item.business_info?.business_name === '未知业务ID' ||
         !item.business_info?.business_name
}

function doSearch() {
  page.value = 1
  fetchData()
}

function doExport() {
  const params: Record<string, string> = {}
  if (searchText.value.trim()) params.business_name = searchText.value.trim()
  if (idcCode.value.trim()) params.idc_code = idcCode.value.trim()
  downloadExport('/business-info/export', params)
}

function goPage(p: number) {
  page.value = p
  fetchData()
}

function formatSpeed(speed: number): string {
  if (!speed || speed === 0) return '-'
  if (speed >= 10000) return (speed / 1000).toFixed(0) + 'G'
  return speed + 'M'
}
</script>

<template>
  <div class="page-header">
    <h2>业务信息</h2>
    <p class="page-hint">按业务名称（如 SPOP_ZX）和机房编码（如 B11）过滤。勾选"显示未知业务"可查看尚未分配业务的机器。</p>
  </div>

  <div class="search-bar">
    <div class="search-field">
      <label>业务名称</label>
      <input v-model="searchText" placeholder="如 SPOP_ZX、百度ANT" @keyup.enter="doSearch" />
    </div>
    <div class="search-field">
      <label>机房编码</label>
      <input v-model="idcCode" placeholder="如 B11" @keyup.enter="doSearch" style="min-width:100px" />
    </div>
    <label style="display:flex;align-items:center;gap:6px;font-size:13px;margin-bottom:2px;cursor:pointer">
      <input type="checkbox" v-model="showUnknown" @change="doSearch" />
      显示未知业务
    </label>
    <button @click="doSearch">查询</button>
    <button style="background:#fff;color:#666;border:1px solid #d9d9d9" @click="searchText='';idcCode='';showUnknown=true;doSearch()">重置</button>
    <button style="background:#52c41a;color:#fff;border:none" @click="doExport">导出Excel</button>
  </div>

  <div class="data-card">
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="filteredList.length === 0" class="empty">暂无数据</div>
    <template v-else>
      <div class="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>ZbxID</th>
              <th>IPMI IP</th>
              <th>机房</th>
              <th>业务名称</th>
              <th>业务ID</th>
              <th>带宽</th>
              <th>旧业务名称</th>
              <th>旧业务ID</th>
              <th>旧带宽</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, idx) in filteredList" :key="idx"
              :class="{ 'row-unknown': isUnknown(item) }">
              <td>{{ item.idc_info?.zbx_id }}</td>
              <td>{{ item.idc_info?.ipmi_ip }}</td>
              <td>{{ item.idc_info?.idc_code }}</td>
              <td>
                <span v-if="isUnknown(item)" class="tag-unknown">未知</span>
                {{ item.business_info?.business_name || '-' }}
              </td>
              <td>{{ item.business_info?.business_id || '-' }}</td>
              <td>{{ formatSpeed(item.business_info?.business_speed) }}</td>
              <td>{{ item.business_info?.old_business_name || '-' }}</td>
              <td>{{ item.business_info?.old_business_id || '-' }}</td>
              <td>{{ formatSpeed(item.business_info?.old_business_speed) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="table-footer">
        <span class="total">共 {{ total }} 条记录（当前显示 {{ filteredList.length }} 条）</span>
        <div class="pagination">
          <button :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
          <template v-for="p in totalPages" :key="p">
            <button v-if="Math.abs(p - page) <= 3 || p === 1 || p === totalPages"
              :class="{ active: p === page }"
              @click="goPage(p)">{{ p }}</button>
          </template>
          <button :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
        </div>
        <span class="page-info">{{ page }} / {{ totalPages }}</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.row-unknown { opacity: 0.55; }
.tag-unknown {
  display: inline-block;
  background: #fff3cd;
  color: #856404;
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 3px;
  margin-right: 4px;
  vertical-align: middle;
}
</style>
