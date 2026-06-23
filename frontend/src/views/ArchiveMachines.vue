<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { downloadExport, getMachineArchive, listMachineArchives, listMachineSyncStates } from '../api'

const filters = ref({
  search: '',
  ipmi_ip: '',
  zbx_id: '',
  idc_code: '',
  status: 'archived',
})
const loading = ref(false)
const detailLoading = ref(false)
const total = ref(0)
const page = ref(1)
const size = ref(50)
const archives = ref<any[]>([])
const staleStates = ref<any[]>([])
const selected = ref<any | null>(null)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size.value)))
const staleCount = computed(() => staleStates.value.length)

onMounted(() => {
  fetchArchives()
  fetchStaleStates()
})

function fmt(value: any) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

function daysLeft(value: string) {
  if (!value) return 0
  return Math.max(0, Math.ceil((new Date(value).getTime() - Date.now()) / 86400000))
}

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

async function fetchArchives() {
  loading.value = true
  try {
    const res = await listMachineArchives(buildParams())
    if (res.code === 200) {
      total.value = res.data?.total || 0
      archives.value = res.data?.list || []
      if (!archives.value.some(item => item.archive_batch_id === selected.value?.batch?.archive_batch_id)) {
        selected.value = null
      }
    }
  } finally {
    loading.value = false
  }
}

async function fetchStaleStates() {
  const res = await listMachineSyncStates({ status: 'stale', page: '1', size: '200' })
  if (res.code === 200) staleStates.value = res.data?.list || []
}

async function openDetail(batchId: string) {
  detailLoading.value = true
  try {
    const res = await getMachineArchive(batchId)
    if (res.code === 200) selected.value = res.data
  } finally {
    detailLoading.value = false
  }
}

function doSearch() {
  page.value = 1
  fetchArchives()
}

function resetFilters() {
  filters.value = { search: '', ipmi_ip: '', zbx_id: '', idc_code: '', status: 'archived' }
  doSearch()
}

function doExport() {
  const params = buildParams()
  delete params.page
  delete params.size
  downloadExport('/machine-archives/export', params)
}

function goPage(p: number) {
  page.value = Math.min(Math.max(1, p), totalPages.value)
  fetchArchives()
}
</script>

<template>
  <div class="grid grid-cols-[1fr_420px] gap-4">
    <div class="space-y-4 min-w-0">
      <section class="grid grid-cols-4 gap-3">
        <div class="panel p-4">
          <div class="text-12px text-ink-500">归档批次</div>
          <div class="mt-2 text-26px font-800 text-ink-950">{{ total }}</div>
        </div>
        <div class="panel p-4">
          <div class="text-12px text-ink-500">待归档</div>
          <div class="mt-2 text-26px font-800 text-ink-950">{{ staleCount }}</div>
        </div>
        <div class="panel p-4">
          <div class="text-12px text-ink-500">保留周期</div>
          <div class="mt-2 text-26px font-800 text-ink-950">30天</div>
        </div>
        <div class="panel p-4">
          <div class="text-12px text-ink-500">归档条件</div>
          <div class="mt-2 text-16px font-800 text-ink-950">超时未更新</div>
        </div>
      </section>

      <section class="panel p-4">
        <div class="grid grid-cols-[1.2fr_1fr_1fr_0.8fr_0.8fr_auto] gap-3 items-end">
          <label>
            <span class="text-12px font-600 text-ink-500">搜索</span>
            <input v-model="filters.search" class="field mt-1 w-full" placeholder="IP / 机房 / SSH" @keyup.enter="doSearch" />
          </label>
          <label>
            <span class="text-12px font-600 text-ink-500">IPMI IP</span>
            <input v-model="filters.ipmi_ip" class="field mt-1 w-full" @keyup.enter="doSearch" />
          </label>
          <label>
            <span class="text-12px font-600 text-ink-500">ZbxID</span>
            <input v-model="filters.zbx_id" class="field mt-1 w-full" @keyup.enter="doSearch" />
          </label>
          <label>
            <span class="text-12px font-600 text-ink-500">机房</span>
            <input v-model="filters.idc_code" class="field mt-1 w-full" @keyup.enter="doSearch" />
          </label>
          <label>
            <span class="text-12px font-600 text-ink-500">状态</span>
            <select v-model="filters.status" class="field mt-1 w-full">
              <option value="archived">归档中</option>
              <option value="restored">已恢复</option>
              <option value="all">全部</option>
            </select>
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
        <div v-else-if="archives.length === 0" class="py-18 text-center text-13px text-ink-500">暂无归档机器</div>
        <template v-else>
          <div class="overflow-auto">
            <table class="min-w-300 w-full border-collapse">
              <thead>
                <tr>
                  <th class="th">IPMI IP</th>
                  <th class="th">ZbxID</th>
                  <th class="th">机房</th>
                  <th class="th">SSH IP</th>
                  <th class="th">磁盘</th>
                  <th class="th">最后更新</th>
                  <th class="th">归档时间</th>
                  <th class="th">剩余</th>
                  <th class="th">状态</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in archives"
                  :key="item.archive_batch_id"
                  class="cursor-pointer hover:bg-[#fbfcff]"
                  :class="selected?.batch?.archive_batch_id === item.archive_batch_id ? 'bg-[#eef4ff]' : ''"
                  @click="openDetail(item.archive_batch_id)"
                >
                  <td class="td font-mono">{{ item.ipmi_ip }}</td>
                  <td class="td max-w-54 truncate font-mono text-12px" :title="item.zbx_id">{{ item.zbx_id }}</td>
                  <td class="td">
                    <div class="font-700 text-ink-900">{{ compact(item.idc_code) }}</div>
                    <div class="mt-0.5 max-w-48 truncate text-12px text-ink-500">{{ compact(item.idc_name) }}</div>
                  </td>
                  <td class="td font-mono">{{ compact(item.ssh_ip) }}</td>
                  <td class="td">
                    {{ compact(item.system_disk) }} / SSD {{ compact(item.ssd_count) }} / HDD {{ compact(item.hdd_count) }} / 系统直通 {{ compact(item.sys_hdd_count) }}
                  </td>
                  <td class="td">{{ fmt(item.last_seen_at) }}</td>
                  <td class="td">{{ fmt(item.archived_at) }}</td>
                  <td class="td">
                    <span class="rounded-full px-2 py-0.5 text-11px" :class="daysLeft(item.expires_at) <= 3 ? 'bg-red-50 text-red-700' : 'bg-emerald-50 text-emerald-700'">
                      {{ daysLeft(item.expires_at) }}天
                    </span>
                  </td>
                  <td class="td">
                    <span class="rounded-full bg-[#eef4ff] px-2 py-0.5 text-11px text-brand-600">{{ item.status }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="border-t border-[#edf1f7] px-4 py-3 flex items-center justify-between">
            <div class="text-12px text-ink-500">共 {{ total }} 个归档批次</div>
            <div class="flex items-center gap-2">
              <button class="btn-soft" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
              <span class="text-13px text-ink-600">{{ page }} / {{ totalPages }}</span>
              <button class="btn-soft" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
            </div>
          </div>
        </template>
      </section>
    </div>

    <aside class="panel sticky top-21 h-[calc(100vh-108px)] overflow-hidden">
      <div class="border-b border-[#edf1f7] px-4 py-3 flex items-center justify-between">
        <div>
          <div class="text-14px font-800 text-ink-950">归档快照</div>
          <div class="mt-0.5 text-12px text-ink-500">四张表的历史字段</div>
        </div>
        <span v-if="detailLoading" class="i-lucide-loader-circle animate-spin text-ink-500" />
      </div>

      <div v-if="!selected" class="h-full flex items-center justify-center px-8 text-center text-13px leading-6 text-ink-500">
        选择左侧归档记录查看 IDC、硬件、业务和网络信息快照。
      </div>

      <div v-else class="h-[calc(100%-58px)] overflow-auto p-4 space-y-4">
        <section class="rounded-2 border border-[#e4e9f1] p-4">
          <div class="mb-3 text-13px font-800 text-ink-950">IDC 信息</div>
          <dl class="grid grid-cols-2 gap-3 text-12px">
            <div><dt class="text-ink-500">IPMI</dt><dd class="mt-1 font-mono">{{ compact(selected.idc_info?.ipmi_ip) }}</dd></div>
            <div><dt class="text-ink-500">SSH</dt><dd class="mt-1 font-mono">{{ compact(selected.idc_info?.ssh_ip) }}</dd></div>
            <div><dt class="text-ink-500">机房编码</dt><dd class="mt-1">{{ compact(selected.idc_info?.idc_code) }}</dd></div>
            <div><dt class="text-ink-500">归档原因</dt><dd class="mt-1">{{ compact(selected.batch?.archive_reason) }}</dd></div>
            <div class="col-span-2"><dt class="text-ink-500">机房名称</dt><dd class="mt-1">{{ compact(selected.idc_info?.idc_name) }}</dd></div>
            <div class="col-span-2"><dt class="text-ink-500">ZbxID</dt><dd class="mt-1 break-all font-mono">{{ compact(selected.idc_info?.zbx_id) }}</dd></div>
          </dl>
        </section>

        <section class="rounded-2 border border-[#e4e9f1] p-4">
          <div class="mb-3 text-13px font-800 text-ink-950">硬件信息</div>
          <dl class="grid grid-cols-2 gap-3 text-12px">
            <div><dt class="text-ink-500">系统</dt><dd class="mt-1">{{ compact(selected.machine_info?.system_type) }}</dd></div>
            <div><dt class="text-ink-500">厂商</dt><dd class="mt-1">{{ compact(selected.machine_info?.manufacturer) }}</dd></div>
            <div><dt class="text-ink-500">内存</dt><dd class="mt-1">{{ compact(selected.machine_info?.memory_count) }}</dd></div>
            <div><dt class="text-ink-500">高度</dt><dd class="mt-1">{{ compact(selected.machine_info?.server_height) }}</dd></div>
            <div><dt class="text-ink-500">系统盘</dt><dd class="mt-1">{{ compact(selected.machine_info?.system_disk) }}</dd></div>
            <div><dt class="text-ink-500">SSD</dt><dd class="mt-1">{{ compact(selected.machine_info?.ssd_count) }}</dd></div>
            <div><dt class="text-ink-500">HDD</dt><dd class="mt-1">{{ compact(selected.machine_info?.hdd_count) }}</dd></div>
            <div><dt class="text-ink-500">系统直通HDD</dt><dd class="mt-1">{{ compact(selected.machine_info?.sys_hdd_count) }}</dd></div>
            <div class="col-span-2"><dt class="text-ink-500">交换机端口</dt><dd class="mt-1 font-mono">{{ compact(selected.machine_info?.switch_port) }}</dd></div>
            <div class="col-span-2"><dt class="text-ink-500">序列号</dt><dd class="mt-1 font-mono">{{ compact(selected.machine_info?.server_sn) }}</dd></div>
            <div class="col-span-2"><dt class="text-ink-500">CPU</dt><dd class="mt-1">{{ compact(selected.machine_info?.cpu_info) }}</dd></div>
          </dl>
        </section>

        <section class="rounded-2 border border-[#e4e9f1] p-4">
          <div class="mb-3 text-13px font-800 text-ink-950">业务信息</div>
          <dl class="grid grid-cols-2 gap-3 text-12px">
            <div><dt class="text-ink-500">业务名</dt><dd class="mt-1">{{ compact(selected.business_info?.business_name) }}</dd></div>
            <div><dt class="text-ink-500">业务ID</dt><dd class="mt-1">{{ compact(selected.business_info?.business_id) }}</dd></div>
            <div><dt class="text-ink-500">带宽</dt><dd class="mt-1">{{ compact(selected.business_info?.business_speed) }}</dd></div>
            <div><dt class="text-ink-500">旧业务</dt><dd class="mt-1">{{ compact(selected.business_info?.old_business_name) }}</dd></div>
          </dl>
        </section>

        <section class="rounded-2 border border-[#e4e9f1] p-4">
          <div class="mb-3 flex items-center justify-between">
            <div class="text-13px font-800 text-ink-950">网络信息</div>
            <span class="rounded-full bg-[#eef4ff] px-2 py-0.5 text-11px text-brand-600">{{ selected.network_info?.length || 0 }} 条</span>
          </div>
          <div class="space-y-2">
            <div v-for="net in selected.network_info || []" :key="net.archive_id" class="rounded-1.5 bg-[#f8fafc] p-3 text-12px">
              <div class="font-800 text-ink-900">{{ compact(net.eth_name) }} / {{ compact(net.net_type) }}</div>
              <div class="mt-1 font-mono text-ink-500">{{ compact(net.ipv4_ip) }} / {{ compact(net.mac_address) }}</div>
              <div class="mt-1 text-ink-500">VLAN {{ compact(net.vlan) }} · {{ compact(net.ip_status) }}</div>
            </div>
          </div>
        </section>
      </div>
    </aside>
  </div>
</template>
