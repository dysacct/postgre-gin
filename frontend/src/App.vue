<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const isLogin = computed(() => route.name === 'Login')
const username = computed(() => localStorage.getItem('username') || 'operator')

const menuItems = [
  { path: '/machine-info', label: '机器信息', icon: 'i-lucide-server' },
  { path: '/archives', label: '归档机器', icon: 'i-lucide-archive' },
  { path: '/network-info', label: '网络信息', icon: 'i-lucide-network' },
  { path: '/idc-info', label: 'SSH信息', icon: 'i-lucide-key-round' },
  { path: '/business-info', label: '业务信息', icon: 'i-lucide-chart-no-axes-combined' },
  { path: '/deletion', label: '删除管理', icon: 'i-lucide-trash-2' },
]

function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  localStorage.removeItem('role')
  router.push('/login')
}
</script>

<template>
  <router-view v-if="isLogin" />

  <div v-else class="page-shell flex">
    <aside class="fixed inset-y-0 left-0 z-20 w-62 border-r border-[#e5ebf3] bg-[#111827] text-white">
      <div class="h-17 border-b border-white/8 px-5 flex items-center">
        <div>
          <div class="text-18px font-700 tracking-wide">CMDB</div>
          <div class="mt-0.5 text-11px text-white/45">Machine Operations</div>
        </div>
      </div>

      <nav class="px-3 py-4 space-y-1">
        <router-link
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="group flex h-10 items-center gap-3 rounded-2 px-3 text-13px text-white/62 transition hover:bg-white/8 hover:text-white"
          active-class="!bg-white !text-[#111827] shadow-sm"
        >
          <span :class="item.icon" class="text-16px" />
          <span>{{ item.label }}</span>
        </router-link>
      </nav>

      <div class="absolute bottom-0 left-0 right-0 border-t border-white/8 p-4">
        <div class="mb-3 flex items-center gap-3">
          <div class="h-8 w-8 rounded-2 bg-white/12 flex items-center justify-center text-12px font-700">
            {{ username.slice(0, 1).toUpperCase() }}
          </div>
          <div class="min-w-0">
            <div class="truncate text-13px font-600">{{ username }}</div>
            <div class="text-11px text-white/45">已登录</div>
          </div>
        </div>
        <button class="btn w-full border border-white/12 bg-white/8 text-white/72 hover:bg-white/14 hover:text-white" @click="logout">
          <span class="i-lucide-log-out" />
          退出登录
        </button>
      </div>
    </aside>

    <main class="ml-62 min-h-screen flex-1">
      <header class="sticky top-0 z-10 h-17 border-b border-[#e5ebf3] bg-white/86 backdrop-blur px-6 flex items-center justify-between">
        <div>
          <div class="text-12px text-ink-500">资产管理系统</div>
          <h1 class="mt-0.5 text-18px font-700 text-ink-950">{{ route.meta.title || '控制台' }}</h1>
        </div>
        <div class="flex items-center gap-2 text-12px text-ink-500">
          <span class="h-2 w-2 rounded-full bg-emerald-500" />
          API 34185
        </div>
      </header>
      <div class="p-6">
        <router-view />
      </div>
    </main>
  </div>
</template>
