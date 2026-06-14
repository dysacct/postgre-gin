<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { searchMachines, listMachines, downloadExport } from '../api'

const idcCode = ref('')
const idcName = ref('')
const ipmiIP = ref('')
const zbxID = ref('')
const searchInput = ref('')
const loading = ref(false)

const total = ref(0)
const page = ref(1)
const size = ref(300)
const list = ref<any[]>([])
const expanded = ref<Set<number>>(new Set())

const totalPages = computed(() => Math.ceil(total.value / size.value) || 1)

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const params: Record<string, string> = { page: String(page.value), size: String(size.value) }

    const idcCodeVal = idcCode.value.trim()
    const idcNameVal = idcName.value.trim()
    const ipmiIPVal = ipmiIP.value.trim()
    const zbxIDVal = zbxID.value.trim()
    const searchVal = searchInput.value.trim()

    // 有任意精确搜索字段 → 使用 searchMachines API (多字段 OR 匹配)
    if (idcCodeVal || idcNameVal || ipmiIPVal || zbxIDVal) {
      if (idcCodeVal) params.idc_code = idcCodeVal
      if (idcNameVal) params.idc_name = idcNameVal
      if (ipmiIPVal) params.ipmi_ip = ipmiIPVal
      if (zbxIDVal) params.zbx_id = zbxIDVal
      const res = await searchMachines(params)
      if (res.code === 200) {
        total.value = res.data?.total || 0
        list.value = res.data?.list || []
      }
    } else if (searchVal) {
      // 仅全局搜索 → 跨表模糊匹配
      const res = await listMachines({ ...params, search: searchVal })
      if (res.code === 200) {
        total.value = res.data?.total || 0
        list.value = res.data?.list || []
      }
    } else {
      // 无搜索条件 → 全量列表
      const res = await listMachines(params)
      if (res.code === 200) {
        total.value = res.data?.total || 0
        list.value = res.data?.list || []
      }
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
  if (idcCode.value.trim()) params.idc_code = idcCode.value.trim()
  if (idcName.value.trim()) params.idc_name = idcName.value.trim()
  if (ipmiIP.value.trim()) params.ipmi_ip = ipmiIP.value.trim()
  if (zbxID.value.trim()) params.zbx_id = zbxID.value.trim()
  if (searchInput.value.trim()) params.search = searchInput.value.trim()
  downloadExport('/machines/export', params)
}

function goPage(p: number) {
  page.value = p
  fetchData()
}

function toggleExpand(idx: number) {
  const s = new Set(expanded.value)
  if (s.has(idx)) s.delete(idx)
  else s.add(idx)
  expanded.value = s
}
</script>

<template>
  <div class="page-header">
    <h2>机器信息</h2>
  </div>

  <div class="search-bar">
    <div class="search-field">
      <label>IPMI IP (单台查询)</label>
      <input v-model="ipmiIP" placeholder="如 10.0.0.1" @keyup.enter="doSearch" style="min-width:130px" />
    </div>
    <div class="search-field">
      <label>ZbxID</label>
      <input v-model="zbxID" placeholder="如 10001" @keyup.enter="doSearch" style="min-width:100px" />
    </div>
    <div class="search-field">
      <label>机房编码 (如 B11)</label>
      <input v-model="idcCode" placeholder="idc_code" @keyup.enter="doSearch" style="min-width:100px" />
    </div>
    <div class="search-field">
      <label>机房名称</label>
      <input v-model="idcName" placeholder="idc_name" @keyup.enter="doSearch" style="min-width:100px" />
    </div>
    <div class="search-field">
      <label>全局搜索</label>
      <input v-model="searchInput" placeholder="IP/业务名/序列号等" @keyup.enter="doSearch" />
    </div>
    <button @click="doSearch">查询</button>
    <button style="background:#fff;color:#666;border:1px solid #d9d9d9" @click="ipmiIP='';zbxID='';idcCode='';idcName='';searchInput='';doSearch()">重置</button>
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
              <th style="width:40px"></th>
              <th>ZbxID</th>
              <th>IPMI IP</th>
              <th>机房编码</th>
              <th>机房名称</th>
              <th>SSH IP</th>
              <th>系统类型</th>
              <th>厂商</th>
              <th>CPU</th>
              <th>内存</th>
              <th>系统盘</th>
              <th>SSD</th>
              <th>HDD</th>
              <th>高度</th>
              <th>序列号</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="(item, idx) in list" :key="idx">
              <tr>
                <td>
                  <span class="detail-toggle" @click="toggleExpand(idx)">
                    {{ expanded.has(idx) ? '▼' : '▶' }}
                  </span>
                </td>
                <td>{{ item.idc_info?.zbx_id }}</td>
                <td>{{ item.idc_info?.ipmi_ip }}</td>
                <td>{{ item.idc_info?.idc_code }}</td>
                <td>{{ item.idc_info?.idc_name }}</td>
                <td>{{ item.idc_info?.ssh_ip }}</td>
                <td>{{ item.machine_info?.system_type }}</td>
                <td>{{ item.machine_info?.manufacturer }}</td>
                <td :title="item.machine_info?.cpu_info">{{ item.machine_info?.cpu_info }}</td>
                <td>{{ item.machine_info?.memory_count }}</td>
                <td>{{ item.machine_info?.system_disk }}</td>
                <td>{{ item.machine_info?.ssd_count }}</td>
                <td>{{ item.machine_info?.hdd_count }}</td>
                <td>{{ item.machine_info?.server_height }}</td>
                <td>{{ item.machine_info?.server_sn }}</td>
              </tr>
              <tr v-if="expanded.has(idx)" class="detail-row">
                <td colspan="15">
                  <div class="detail-panel">
                    <div class="detail-grid">
                      <template v-if="item.business_info?.business_name">
                        <div class="detail-item"><strong>业务名:</strong>{{ item.business_info?.business_name }}</div>
                        <div class="detail-item"><strong>业务ID:</strong>{{ item.business_info?.business_id }}</div>
                        <div class="detail-item"><strong>业务带宽:</strong>{{ item.business_info?.business_speed }}M</div>
                        <div class="detail-item"><strong>旧业务名:</strong>{{ item.business_info?.old_business_name }}</div>
                      </template>
                      <template v-if="item.network_info?.length">
                        <div class="detail-item"><strong>网卡数:</strong>{{ item.network_info.length }}</div>
                        <div v-for="(net, ni) in item.network_info" :key="ni" class="detail-item">
                          <strong>网卡{{ ni+1 }}:</strong>{{ net.eth_name }} / {{ net.ipv4_ip }} / {{ net.mac_address }}
                        </div>
                      </template>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
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
