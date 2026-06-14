<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { deleteMachines, deleteNetworks, getDeletedRecords, downloadExport } from '../api'

const activeTab = ref<'machine' | 'network'>('machine')
const loading = ref(false)
const machineIdcCode = ref('')
const machineIPMIs = ref('')
const networkIdcCode = ref('')
const networkIPMIs = ref('')
const showModal = ref(false)
const modalType = ref<'machine' | 'network'>('machine')
const modalPreview = ref<any[]>([])
const records = ref<any[]>([])
const recordsTotal = ref(0)
const recordsPage = ref(1)

// 机器tab: 按ipmi_ip合并idc_info为一行
const mergedMachines = computed(() => {
  const groups: Record<string, { idc: any; deletedAt: string; deletedBy: string; expiresAt: string }> = {}
  for (const rec of records.value) {
    const key = rec.ipmi_ip || `rec-${rec.id}`
    if (!groups[key]) groups[key] = { idc: null, deletedAt: rec.deleted_at, deletedBy: rec.deleted_by, expiresAt: rec.expires_at }
    if (rec.source_table === 'idc_info') groups[key].idc = parseData(rec.record_data)
  }
  return Object.entries(groups).map(([ipmi, g]) => ({
    ipmi,
    zbxId: g.idc?.zbx_id,
    idcCode: g.idc?.idc_code,
    idcName: g.idc?.idc_name,
    sshIp: g.idc?.ssh_ip,
    deletedAt: g.deletedAt,
    deletedBy: g.deletedBy,
    expiresAt: g.expiresAt,
  }))
})

// 网络tab: 每条network_info记录一行
const networkRows = computed(() => records.value.map(rec => {
  const d = parseData(rec.record_data)
  return {
    id: rec.id,
    ipmiIp: d.ipmi_ip || rec.ipmi_ip,
    ipv4Ip: d.ipv4_ip,
    ipv6Ip: d.ipv6_ip,
    macAddress: d.mac_address,
    ethName: d.eth_name,
    idcCode: d.idc_code,
    netType: d.net_type,
    vlan: d.vlan,
    gateway: d.ipv4_gateway,
    ipSpeed: d.ip_speed,
    ipStatus: d.ip_status,
    ipNotes: d.ip_notes,
    deletedAt: rec.deleted_at,
    deletedBy: rec.deleted_by,
    expiresAt: rec.expires_at,
  }
}))

onMounted(() => fetchRecords())

async function fetchRecords() {
  try {
    const res = await getDeletedRecords({ page: '1', size: '2000', record_type: activeTab.value })
    if (res.code === 200) {
      records.value = res.data?.list || []
      recordsTotal.value = res.data?.total || 0
    }
  } catch {}
}

function openModal(type: 'machine' | 'network') {
  modalType.value = type
  if (type === 'machine') {
    const idc = machineIdcCode.value.trim()
    const ips = machineIPMIs.value.split(/[\n,]/).map(s => s.trim()).filter(s => s.length > 0)
    if (!idc && ips.length === 0) { alert('请填写机房编码或 IPMI IP'); return }
    modalPreview.value = idc ? [{ label: '机房编码', value: idc }] : ips.map(ip => ({ label: 'IPMI IP', value: ip }))
  } else {
    const idc = networkIdcCode.value.trim()
    const ips = networkIPMIs.value.split(/[\n,]/).map(s => s.trim()).filter(s => s.length > 0)
    if (!idc && ips.length === 0) { alert('请填写机房编码或 IPMI IP'); return }
    modalPreview.value = idc ? [{ label: '机房编码', value: idc }] : ips.map(ip => ({ label: 'IPMI IP', value: ip }))
  }
  showModal.value = true
}

async function confirmDelete() {
  showModal.value = false
  loading.value = true
  try {
    if (modalType.value === 'machine') {
      const idc = machineIdcCode.value.trim()
      const ips = machineIPMIs.value.split(/[\n,]/).map(s => s.trim()).filter(s => s.length > 0)
      const res = await deleteMachines({ idc_code: idc || undefined, ipmi_ips: ips.length > 0 ? ips : undefined })
      if (res.code === 200) { alert(res.message); machineIdcCode.value = ''; machineIPMIs.value = ''; fetchRecords() }
      else { alert('删除失败: ' + res.message) }
    } else {
      const idc = networkIdcCode.value.trim()
      const ips = networkIPMIs.value.split(/[\n,]/).map(s => s.trim()).filter(s => s.length > 0)
      const res = await deleteNetworks({ idc_code: idc || undefined, ipmi_ips: ips.length > 0 ? ips : undefined })
      if (res.code === 200) { alert(res.message); networkIdcCode.value = ''; networkIPMIs.value = ''; fetchRecords() }
      else { alert('删除失败: ' + res.message) }
    }
  } catch (e: any) { alert('请求失败: ' + e.message) }
  finally { loading.value = false }
}

function switchTab(tab: 'machine' | 'network') {
  activeTab.value = tab
  recordsPage.value = 1
  fetchRecords()
}

function daysLeft(expiresAt: string): number {
  return Math.max(0, Math.ceil((new Date(expiresAt).getTime() - Date.now()) / (1000 * 60 * 60 * 24)))
}

function parseData(data: string): any {
  try { return JSON.parse(data) } catch { return {} }
}

function doExport() {
  downloadExport('/deletion/records/export', { record_type: activeTab.value })
}
</script>

<template>
  <div class="page-header">
    <h2>删除管理</h2>
    <p class="page-hint">删除的数据将归档保留30天，到期后自动清理。删除操作不可撤销，请谨慎操作。</p>
  </div>

  <div class="deletion-panels">
    <div class="deletion-card" :class="{ active: activeTab === 'machine' }" @click="switchTab('machine')">
      <h3>删除机器信息（三表）</h3>
      <p>删除 idc_info + machine_info + business_info</p>
      <div v-if="activeTab === 'machine'" class="deletion-form" @click.stop>
        <div class="form-group">
          <label>按机房编码删除</label>
          <input v-model="machineIdcCode" placeholder="如 B11" :disabled="!!machineIPMIs.trim()" />
        </div>
        <div class="form-group">
          <label>按 IPMI IP 删除（多个用逗号或换行分隔）</label>
          <textarea v-model="machineIPMIs" placeholder="10.0.0.1&#10;10.0.0.2" rows="4" :disabled="!!machineIdcCode.trim()"></textarea>
        </div>
        <button class="btn-danger" :disabled="loading" @click="openModal('machine')">{{ loading ? '删除中...' : '删除机器信息' }}</button>
      </div>
    </div>

    <div class="deletion-card" :class="{ active: activeTab === 'network' }" @click="switchTab('network')">
      <h3>删除网络信息（单表）</h3>
      <p>仅删除 network_info 表中的记录</p>
      <div v-if="activeTab === 'network'" class="deletion-form" @click.stop>
        <div class="form-group">
          <label>按机房编码删除</label>
          <input v-model="networkIdcCode" placeholder="如 B11" :disabled="!!networkIPMIs.trim()" />
        </div>
        <div class="form-group">
          <label>按 IPMI IP 删除</label>
          <textarea v-model="networkIPMIs" placeholder="10.0.0.1&#10;10.0.0.2" rows="4" :disabled="!!networkIdcCode.trim()"></textarea>
        </div>
        <button class="btn-danger" :disabled="loading" @click="openModal('network')">{{ loading ? '删除中...' : '删除网络信息' }}</button>
      </div>
    </div>
  </div>

  <!-- 确认弹框 -->
  <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
    <div class="modal-box">
      <div class="modal-header"><h3>确认删除</h3></div>
      <div class="modal-body">
        <p class="modal-warn">此操作不可撤销！将删除以下内容并归档30天：</p>
        <div class="modal-list">
          <div v-for="(item, idx) in modalPreview" :key="idx" class="modal-item">
            <span class="modal-label">{{ item.label }}：</span><code>{{ item.value }}</code>
          </div>
        </div>
        <p class="modal-type">删除类型：<strong>{{ modalType === 'machine' ? '机器信息（三表）' : '网络信息（单表）' }}</strong></p>
      </div>
      <div class="modal-footer">
        <button class="btn-cancel" @click="showModal = false">取消</button>
        <button class="btn-danger" :disabled="loading" @click="confirmDelete">{{ loading ? '删除中...' : '确认删除' }}</button>
      </div>
    </div>
  </div>

  <!-- 机器删除记录表格 -->
  <div v-if="activeTab === 'machine'" class="data-card" style="margin-top:20px">
    <div class="table-header-bar">
      <h3 style="margin:0;font-size:15px">已删除机器信息（保留30天）</h3>
      <div style="display:flex;gap:8px;align-items:center">
        <span style="font-size:13px;color:#999">共 {{ mergedMachines.length }} 台</span>
        <button style="background:#52c41a;color:#fff;border:none;padding:6px 14px;border-radius:4px;font-size:12px;cursor:pointer" @click="doExport">导出Excel</button>
      </div>
    </div>

    <div v-if="mergedMachines.length === 0" class="empty" style="padding:30px">暂无删除记录</div>
    <template v-else>
      <div class="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>ZbxID</th>
              <th>IPMI IP</th>
              <th>机房编码</th>
              <th>机房名称</th>
              <th>SSH IP</th>
              <th>操作人</th>
              <th>删除时间</th>
              <th>剩余天数</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in mergedMachines" :key="m.ipmi">
              <td>{{ m.zbxId || '-' }}</td>
              <td><code>{{ m.ipmi }}</code></td>
              <td>{{ m.idcCode || '-' }}</td>
              <td>{{ m.idcName || '-' }}</td>
              <td>{{ m.sshIp || '-' }}</td>
              <td>{{ m.deletedBy || '-' }}</td>
              <td>{{ m.deletedAt ? new Date(m.deletedAt).toLocaleString('zh-CN') : '-' }}</td>
              <td><span :class="daysLeft(m.expiresAt) <= 3 ? 'days-warn' : 'days-ok'">{{ daysLeft(m.expiresAt) }} 天</span></td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>

  <!-- 网络删除记录表格 -->
  <div v-if="activeTab === 'network'" class="data-card" style="margin-top:20px">
    <div class="table-header-bar">
      <h3 style="margin:0;font-size:15px">已删除网络信息（保留30天）</h3>
      <div style="display:flex;gap:8px;align-items:center">
        <span style="font-size:13px;color:#999">共 {{ networkRows.length }} 条</span>
        <button style="background:#52c41a;color:#fff;border:none;padding:6px 14px;border-radius:4px;font-size:12px;cursor:pointer" @click="doExport">导出Excel</button>
      </div>
    </div>

    <div v-if="networkRows.length === 0" class="empty" style="padding:30px">暂无删除记录</div>
    <template v-else>
      <div class="table-wrapper">
        <table>
          <thead>
            <tr>
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
              <th>操作人</th>
              <th>删除时间</th>
              <th>剩余天数</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="n in networkRows" :key="n.id">
              <td><code>{{ n.ipmiIp || '-' }}</code></td>
              <td>{{ n.ipv4Ip || '-' }}</td>
              <td>{{ n.ipv6Ip || '-' }}</td>
              <td>{{ n.macAddress || '-' }}</td>
              <td>{{ n.ethName || '-' }}</td>
              <td>{{ n.idcCode || '-' }}</td>
              <td>{{ n.netType || '-' }}</td>
              <td>{{ n.vlan || '-' }}</td>
              <td>{{ n.gateway || '-' }}</td>
              <td>{{ n.ipSpeed || '-' }}</td>
              <td>{{ n.ipStatus || '-' }}</td>
              <td :title="n.ipNotes">{{ (n.ipNotes || '-').substring(0, 20) }}{{ (n.ipNotes || '').length > 20 ? '...' : '' }}</td>
              <td>{{ n.deletedBy || '-' }}</td>
              <td>{{ n.deletedAt ? new Date(n.deletedAt).toLocaleString('zh-CN') : '-' }}</td>
              <td><span :class="daysLeft(n.expiresAt) <= 3 ? 'days-warn' : 'days-ok'">{{ daysLeft(n.expiresAt) }} 天</span></td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<style scoped>
.deletion-panels { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.deletion-card { background: #fff; border: 2px solid #e8e8e8; border-radius: 10px; padding: 24px; cursor: pointer; transition: border-color 0.2s; }
.deletion-card:hover { border-color: #d0d0d0; }
.deletion-card.active { border-color: #ff4d4f; }
.deletion-card h3 { font-size: 16px; margin: 0 0 4px; color: #333; }
.deletion-card > p { font-size: 13px; color: #999; margin: 0 0 12px; }
.deletion-form { margin-top: 16px; }
.deletion-form .form-group { margin-bottom: 14px; }
.deletion-form label { display: block; font-size: 13px; color: #666; margin-bottom: 4px; }
.deletion-form input, .deletion-form textarea { width: 100%; padding: 8px 12px; border: 1px solid #d9d9d9; border-radius: 6px; font-size: 13px; font-family: inherit; }
.deletion-form input:focus, .deletion-form textarea:focus { outline: none; border-color: #ff4d4f; }
.btn-danger { padding: 10px 24px; background: #ff4d4f; color: #fff; border: none; border-radius: 6px; font-size: 14px; cursor: pointer; }
.btn-danger:hover:not(:disabled) { background: #e63946; }
.btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.45); display: flex; align-items: center; justify-content: center; z-index: 2000; }
.modal-box { background: #fff; border-radius: 12px; width: 480px; max-width: 90vw; box-shadow: 0 8px 40px rgba(0,0,0,0.2); }
.modal-header { padding: 20px 24px 0; }
.modal-header h3 { font-size: 18px; }
.modal-body { padding: 16px 24px; }
.modal-warn { color: #ff4d4f; font-size: 14px; margin: 0 0 16px; }
.modal-list { background: #fafafa; border-radius: 8px; padding: 12px 16px; margin-bottom: 16px; max-height: 200px; overflow-y: auto; }
.modal-item { padding: 6px 0; font-size: 14px; border-bottom: 1px solid #f0f0f0; }
.modal-item:last-child { border-bottom: none; }
.modal-label { color: #999; }
.modal-item code { background: #fff; padding: 2px 8px; border-radius: 4px; font-size: 13px; color: #333; }
.modal-type { font-size: 13px; color: #666; }
.modal-footer { display: flex; justify-content: flex-end; gap: 12px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 24px; background: #fff; color: #666; border: 1px solid #d9d9d9; border-radius: 6px; font-size: 14px; cursor: pointer; }
.btn-cancel:hover { border-color: #999; }
.table-header-bar { display: flex; justify-content: space-between; align-items: center; padding: 14px 20px; border-bottom: 1px solid #f0f0f0; }
.days-ok { color: #52c41a; font-weight: 600; }
.days-warn { color: #ff4d4f; font-weight: 600; }
@media (max-width: 768px) { .deletion-panels { grid-template-columns: 1fr; } }
</style>
