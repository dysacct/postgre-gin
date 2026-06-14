<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const isLogin = computed(() => route.name === 'Login')

const username = computed(() => localStorage.getItem('username') || '')

const menuItems = [
  { path: '/machine-info', label: '机器信息', icon: '🖥' },
  { path: '/network-info', label: '网络信息', icon: '🌐' },
  { path: '/idc-info',     label: 'SSH信息',  icon: '🔑' },
  { path: '/business-info',label: '业务信息', icon: '📊' },
  { path: '/deletion',     label: '删除管理', icon: '🗑' },
]

function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  localStorage.removeItem('role')
  router.push('/login')
}
</script>

<template>
  <div v-if="isLogin" class="login-layout">
    <router-view />
  </div>
  <div v-else class="app-layout">
    <aside class="sidebar">
      <div class="sidebar-header">
        <h2>CMDB</h2>
        <span class="subtitle">资产管理系统</span>
      </div>
      <nav class="sidebar-nav">
        <router-link
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          active-class="nav-item--active"
        >
          <span class="nav-icon">{{ item.icon }}</span>
          <span class="nav-label">{{ item.label }}</span>
        </router-link>
      </nav>
      <div class="sidebar-footer">
        <span class="user-badge">{{ username }}</span>
        <button class="logout-btn" @click="logout">退出</button>
      </div>
    </aside>
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>
