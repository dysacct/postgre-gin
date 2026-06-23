<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { downloadExport, listMachines, searchMachines } from '../api'

const filters = ref({
  search: '',
  ipmi_ip: '',
  zbx_id: '',
  idc_code: '',
  idc_name: '',
})
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const size = ref(300)
const list = ref<any[]>([])
const expanded = ref<Set<number>>(new Set())

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size.value)))
const idcCount = computed(() => new Set(list.value.map(item => item.idc_info?.idc_code).filter(Boolean)).size)
const networkCount = computed(() => list.value.reduce((sum, item) => sum + (item.network_info?.length || 0), 0))

onMounted(fetchData)

function compact(value: any) {
  return value === undefined || value === null || value === '' ? '-' : value
}

function buildParams() {
  const params: Record<string, string> = { page: String(page.value), size: String(size.value) }
  for (const [key, value] of Object.entries(filters.value)) {
    const trimmed = value.trim()
    if (trimmed) params[key] = trimmed
  }
  return params
}

async function fetchData() {
  loading.value = true
  try {
    const params = buildParams()
    const hasStructured = !!(params.ipmi_ip || params.zbx_id || params.idc_code || params.idc_name)
    const res = hasStructured ? await searchMachines(params) : await listMachines(params)
    if (res.code === 200) {
      total.value = res.data?.total || 0
      list.value = res.data?.list || []
      expanded.value = new Set()
    }
  } finally {
    loading.value = false
  }
}

function doSearch() {
  page.value = 1
  fetchData()
}

function resetFilters() {
  filters.value = { search: '', ipmi_ip: '', zbx_id: '', idc_code: '', idc_name: '' }
  doSearch()
}

function doExport() {
  const params = buildParams()
  delete params.page
  delete params.size
  downloadExport('/machines/export', params)
}

function goPage(p: number) {
  page.value = Math.min(Math.max(1, p), totalPages.value)
  fetchData()
}

function toggleExpand(idx: number) {
  const next = new Set(expanded.value)
  next.has(idx) ? next.delete(idx) : next.add(idx)
  expanded.value = next
}
</script>

<template>
  <div class="space-y-4">
    <section class="grid grid-cols-4 gap-3">
      <div class="panel p-4">
        <div class="text-12px text-ink-500">当前结果</div>
        <div class="mt-2 text-26px font-800 text-ink-950">{{ total }}</div>
      </div>
      <div class="panel p-4">
        <div class="text-12px text-ink-500">本页机房</div>
        <div class="mt-2 text-26px font-800 text-ink-950">{{ idcCount }}</div>
      </div>
      <div class="panel p-4">
        <div class="text-12px text-ink-500">本页网卡</div>
        <div class="mt-2 text-26px font-800 text-ink-950">{{ networkCount }}</div>
      </div>
      <div class="panel p-4">
        <div class="text-12px text-ink-500">分页</div>
        <div class="mt-2 text-26px font-800 text-ink-950">{{ page }} / {{ totalPages }}</div>
      </div>
    </section>

    <section class="panel p-4">
      <div class="grid grid-cols-[1.2fr_1fr_1fr_0.8fr_1fr_auto] gap-3 items-end">
        <label class="block">
          <span class="text-12px font-600 text-ink-500">全局搜索</span>
          <input v-model="filters.search" class="field mt-1 w-full" placeholder="IP / 业务名 / 序列号" @keyup.enter="doSearch" />
        </label>
        <label class="block">
          <span class="text-12px font-600 text-ink-500">IPMI IP</span>
          <input v-model="filters.ipmi_ip" class="field mt-1 w-full" placeholder="11.96.17.1" @keyup.enter="doSearch" />
        </label>
        <label class="block">
          <span class="text-12px font-600 text-ink-500">ZbxID</span>
          <input v-model="filters.zbx_id" class="field mt-1 w-full" placeholder="ipmi-..." @keyup.enter="doSearch" />
        </label>
        <label class="block">
          <span class="text-12px font-600 text-ink-500">机房编码</span>
          <input v-model="filters.idc_code" class="field mt-1 w-full" placeholder="B96" @keyup.enter="doSearch" />
        </label>
        <label class="block">
          <span class="text-12px font-600 text-ink-500">机房名称</span>
          <input v-model="filters.idc_name" class="field mt-1 w-full" placeholder="湖南移动" @keyup.enter="doSearch" />
        </label>
        <div class="flex gap-2">
          <button class="btn-primary" @click="doSearch"><span class="i-lucide-search" />查询</button>
          <button class="btn-soft" @click="resetFilters"><span class="i-lucide-rotate-ccw" />重置</button>
          <button class="btn-soft" @click="doExport"><span class="i-lucide-download" />导出</button>
        </div>
      </div>
    </section>

    <section class="panel overflow-hidden">
      <div v-if="loading" class="py-18 text-center text-13px text-ink-500">
        <span class="i-lucide-loader-circle animate-spin mr-2" />加载中...
      </div>
      <div v-else-if="list.length === 0" class="py-18 text-center text-13px text-ink-500">暂无数据</div>
      <template v-else>
        <div class="overflow-auto">
          <table class="min-w-360 w-full border-collapse">
            <thead>
              <tr>
                <th class="th w-10"></th>
                <th class="th">ZbxID</th>
                <th class="th">IPMI IP</th>
                <th class="th">机房</th>
                <th class="th">SSH IP</th>
                <th class="th">系统</th>
                <th class="th">厂商</th>
                <th class="th">CPU</th>
                <th class="th">内存</th>
                <th class="th">磁盘</th>
                <th class="th">高度</th>
                <th class="th">交换机端口</th>
                <th class="th">序列号</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="(item, idx) in list" :key="idx">
                <tr class="hover:bg-[#fbfcff]">
                  <td class="td">
                    <button class="h-7 w-7 rounded-1.5 hover:bg-[#eef3fb]" @click="toggleExpand(idx)">
                      <span :class="expanded.has(idx) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" />
                    </button>
                  </td>
                  <td class="td font-mono text-12px">{{ compact(item.idc_info?.zbx_id) }}</td>
                  <td class="td font-mono">{{ compact(item.idc_info?.ipmi_ip) }}</td>
                  <td class="td">
                    <div class="font-600 text-ink-900">{{ compact(item.idc_info?.idc_code) }}</div>
                    <div class="mt-0.5 max-w-56 truncate text-12px text-ink-500">{{ compact(item.idc_info?.idc_name) }}</div>
                  </td>
                  <td class="td font-mono">{{ compact(item.idc_info?.ssh_ip) }}</td>
                  <td class="td">{{ compact(item.machine_info?.system_type) }}</td>
                  <td class="td">{{ compact(item.machine_info?.manufacturer) }}</td>
                  <td class="td max-w-72 truncate" :title="item.machine_info?.cpu_info">{{ compact(item.machine_info?.cpu_info) }}</td>
                  <td class="td">{{ compact(item.machine_info?.memory_count) }}</td>
                  <td class="td">{{ compact(item.machine_info?.system_disk) }} / SSD {{ compact(item.machine_info?.ssd_count) }} / HDD {{ compact(item.machine_info?.hdd_count) }} / 系统直通 {{ compact(item.machine_info?.sys_hdd_count) }}</td>
                  <td class="td">{{ compact(item.machine_info?.server_height) }}</td>
                  <td class="td font-mono">{{ compact(item.machine_info?.switch_port) }}</td>
                  <td class="td font-mono">{{ compact(item.machine_info?.server_sn) }}</td>
                </tr>
                <tr v-if="expanded.has(idx)">
                  <td colspan="13" class="border-t border-[#edf1f7] bg-[#f8fafc] p-4">
                    <div class="grid grid-cols-[1fr_2fr] gap-4">
                      <div class="rounded-2 border border-[#e4e9f1] bg-white p-4">
                        <div class="mb-3 text-13px font-700 text-ink-950">业务信息</div>
                        <div class="grid grid-cols-2 gap-3 text-12px">
                          <div><span class="text-ink-500">业务名</span><div class="mt-1 text-ink-900">{{ compact(item.business_info?.business_name) }}</div></div>
                          <div><span class="text-ink-500">业务ID</span><div class="mt-1 text-ink-900">{{ compact(item.business_info?.business_id) }}</div></div>
                          <div><span class="text-ink-500">带宽</span><div class="mt-1 text-ink-900">{{ compact(item.business_info?.business_speed) }}</div></div>
                          <div><span class="text-ink-500">旧业务</span><div class="mt-1 text-ink-900">{{ compact(item.business_info?.old_business_name) }}</div></div>
                        </div>
                      </div>
                      <div class="rounded-2 border border-[#e4e9f1] bg-white p-4">
                        <div class="mb-3 flex items-center justify-between">
                          <div class="text-13px font-700 text-ink-950">网络信息</div>
                          <span class="rounded-full bg-[#eef4ff] px-2 py-0.5 text-11px text-brand-600">{{ item.network_info?.length || 0 }} 条</span>
                        </div>
                        <div class="grid grid-cols-2 gap-2 text-12px">
                          <div v-for="(net, ni) in item.network_info || []" :key="ni" class="rounded-1.5 bg-[#f8fafc] px-3 py-2">
                            <div class="font-700 text-ink-900">{{ compact(net.eth_name) }} / {{ compact(net.net_type) }}</div>
                            <div class="mt-1 font-mono text-ink-500">{{ compact(net.ipv4_ip) }} / {{ compact(net.mac_address) }}</div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
        <div class="border-t border-[#edf1f7] px-4 py-3 flex items-center justify-between">
          <div class="text-12px text-ink-500">共 {{ total }} 条记录</div>
          <div class="flex items-center gap-2">
            <button class="btn-soft" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
            <span class="text-13px text-ink-600">{{ page }} / {{ totalPages }}</span>
            <button class="btn-soft" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
          </div>
        </div>
      </template>
    </section>
  </div>
</template>
