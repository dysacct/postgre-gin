const BASE = ''

function getToken(): string | null {
  return localStorage.getItem('token')
}

async function request<T = any>(url: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${BASE}${url}`, { ...options, headers })
  if (res.status === 401) {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('role')
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }
  return res.json()
}

// ---- Auth ----
export function login(username: string, password: string) {
  return request('/api/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}

// ---- Machine Info (机器信息) ----
export function searchMachines(params: Record<string, string>) {
  const qs = new URLSearchParams(params).toString()
  return request(`/api/machines/search?${qs}`)
}

export function listMachines(params: Record<string, string>) {
  const qs = new URLSearchParams(params).toString()
  return request(`/api/machines?${qs}`)
}

export function getMachine(ipmiIp: string) {
  return request(`/api/machine/${encodeURIComponent(ipmiIp)}`)
}

// ---- Network Info (网络信息) ----
export function listNetworkInfo(params: Record<string, string>) {
  const qs = new URLSearchParams(params).toString()
  return request(`/api/network-info?${qs}`)
}

export function searchNetworkInfo(params: Record<string, string>) {
  const qs = new URLSearchParams(params).toString()
  return request(`/api/network-info/search?${qs}`)
}

export function getNetworkInfoStats() {
  return request('/api/network-info/stats')
}

// ---- IDC Info (SSH信息) ----
export function getIDCInfo() {
  return request('/api/idc_info')
}

// ---- Deletion (删除管理) ----
export function deleteMachines(data: { idc_code?: string; ipmi_ips?: string[] }) {
  return request('/api/deletion/machines', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function deleteNetworks(data: { idc_code?: string; ipmi_ips?: string[] }) {
  return request('/api/deletion/networks', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function getDeletedRecords(params: Record<string, string>) {
  const qs = new URLSearchParams(params).toString()
  return request(`/api/deletion/records?${qs}`)
}

// ---- Export (导出Excel) ----
export function downloadExport(endpoint: string, params: Record<string, string>) {
  const token = getToken()
  const qs = new URLSearchParams(params).toString()
  const url = `/api${endpoint}?${qs}`
  const a = document.createElement('a')
  a.href = url
  // Add token via query param since <a> download can't set headers
  // Instead, fetch with headers and trigger download
  fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  }).then(res => {
    if (res.status === 401) {
      localStorage.clear()
      window.location.href = '/login'
      return
    }
    return res.blob()
  }).then(blob => {
    if (!blob) return
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = ''
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  })
}
