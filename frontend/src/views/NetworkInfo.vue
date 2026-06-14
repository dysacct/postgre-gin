<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listNetworkInfo, searchNetworkInfo, getNetworkInfoStats, downloadExport } from '../api'

const filters = ref({
  idc_code: '',
  ip_status: '',
  net_type: '',
  ipmi_ip: '',
  zbx_id: '',
  ipv4_ip: '',
  mac_address: '',
})
const stats = ref<any>(null)

const loading = ref(false)
const total = ref(0)
const page = ref(1)
const size = ref(300)
const list = ref<any[]>([])

const totalPages = computed(() => Math.ceil(total.value / size.value) || 1)

onMounted(() => {
  fetchStats()
  fetchData()
})

async function fetchStats() {
  try {
    const res = await getNetworkInfoStats()
    if (res.code === 200) stats.value = res.data
  } catch {}
}

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, string> = { page: String(page.value), size: String(size.value) }
    let hasFilter = false
    for (const [k, v] of Object.entries(filters.value)) {
      if (v.trim()) { params[k] = v.trim(); hasFilter = true }
    }

    let res
    if (hasFilter) {
      res = await searchNetworkInfo(params)
    } else {
      res = await listNetworkInfo(params)
    }
    if (res.code === 200) {
      total.value = res.data?.total || 0
      list.value = res.data?.list || []
    }
  } finally {
    loading.value = false
  }
}

function doSearch() {
  page.value = 1
  fetchData()
}

function doExport() {
  const params: Record<string, string> = {}
  for (const [k, v] of Object.entries(filters.value)) {
    if (v.trim()) params[k] = v.trim()
  }
  downloadExport('/network-info/export', params)
}

function goPage(p: number) {
  page.value = p
  fetchData()
}
</script>

<template>
  <div class="page-header">
    <h2>网络信息</h2>
  </div>

  <div v-if="stats" class="stats-row">
    <div class="stat-card">
      <div class="stat-value">{{ stats.total_count }}</div>
      <div class="stat-label">网络记录总数</div>
      <div class="stat-sub">全量</div>
    </div>
    <div v-for="s in stats.idc_stats?.slice(0,8)" :key="s.idc_code" class="stat-card">
      <div class="stat-value">{{ s.count }}</div>
      <div class="stat-label">机房 {{ s.idc_code }}</div>
      <div class="stat-sub">网络记录数</div>
    </div>
  </div>

  <div class="search-bar">
    <div class="search-field">
      <label>机房编码</label>
      <input v-model="filters.idc_code" placeholder="idc_code" @keyup.enter="doSearch" />
    </div>
    <div class="search-field">
      <label>IP状态</label>
      <select v-model="filters.ip_status">
        <option value="">全部</option>
        <option value="成功">成功</option>
        <option value="失败">失败</option>
        <option value="未启用">未启用</option>
      </select>
    </div>
    <div class="search-field">
      <label>网络类型</label>
      <select v-model="filters.net_type">
        <option value="">全部</option>
        <option value="static">static</option>
        <option value="pppoe">pppoe</option>
      </select>
    </div>
    <div class="search-field">
      <label>IPMI IP</label>
      <input v-model="filters.ipmi_ip" placeholder="ipmi_ip" @keyup.enter="doSearch" />
    </div>
    <div class="search-field">
      <label>IPv4</label>
      <input v-model="filters.ipv4_ip" placeholder="ipv4_ip" @keyup.enter="doSearch" />
    </div>
    <div class="search-field">
      <label>MAC</label>
      <input v-model="filters.mac_address" placeholder="mac_address" @keyup.enter="doSearch" />
    </div>
    <div class="search-field">
      <label>ZbxID</label>
      <input v-model="filters.zbx_id" placeholder="zbx_id" @keyup.enter="doSearch" />
    </div>
    <button @click="doSearch">查询</button>
    <button style="background:#fff;color:#666;border:1px solid #d9d9d9" @click="Object.keys(filters).forEach(k => (filters as any)[k]=''); doSearch()">重置</button>
    <button style="background:#52c41a;color:#fff;border:none" @click="doExport">导出Excel</button>
  </div>

  <div class="data-card">
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="list.length === 0" class="empty">暂无数据</div>
    <template v-else>
      <div class="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>ZbxID</th>
              <th>IPMI IP</th>
              <th>IPv4</th>
              <th>IPv6</th>
              <th>MAC</th>
              <th>网卡</th>
              <th>机房</th>
              <th>网络类型</th>
              <th>VLAN</th>
              <th>网关</th>
              <th>速率</th>
              <th>状态</th>
              <th>备注</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in list" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.zbx_id }}</td>
              <td>{{ item.ipmi_ip }}</td>
              <td>{{ item.ipv4_ip }}</td>
              <td>{{ item.ipv6_ip }}</td>
              <td>{{ item.mac_address }}</td>
              <td>{{ item.eth_name }}</td>
              <td>{{ item.idc_code }}</td>
              <td>{{ item.net_type }}</td>
              <td>{{ item.vlan }}</td>
              <td>{{ item.ipv4_gateway }}</td>
              <td>{{ item.ip_speed }}</td>
              <td>{{ item.ip_status }}</td>
              <td :title="item.ip_notes">{{ item.ip_notes?.substring(0,20) }}{{ item.ip_notes?.length > 20 ? '...' : '' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="table-footer">
        <span class="total">共 {{ total }} 条记录</span>
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
