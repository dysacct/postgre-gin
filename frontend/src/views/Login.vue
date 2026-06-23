<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api'

const router = useRouter()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function doLogin() {
  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await login(username.value, password.value)
    if (res.code === 200 && res.data?.token) {
      localStorage.setItem('token', res.data.token)
      localStorage.setItem('username', res.data.username)
      localStorage.setItem('role', res.data.role)
      router.push('/')
    } else {
      error.value = res.error || res.message || '登录失败'
    }
  } catch (e: any) {
    error.value = '网络错误: ' + e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="min-h-screen bg-[#111827] text-white grid grid-cols-[1.05fr_0.95fr]">
    <section class="relative flex flex-col justify-between overflow-hidden p-10">
      <div class="absolute inset-0 opacity-25 bg-[radial-gradient(circle_at_20%_20%,#60a5fa,transparent_32%),radial-gradient(circle_at_70%_70%,#22c55e,transparent_28%)]" />
      <div class="relative">
        <div class="inline-flex h-10 items-center rounded-2 border border-white/12 bg-white/8 px-3 text-13px text-white/72">
          CMDB Operations Console
        </div>
      </div>
      <div class="relative max-w-145">
        <h1 class="text-44px font-800 leading-tight tracking-normal">机器资产、网络信息和归档状态，放在一个清楚的工作台里。</h1>
        <p class="mt-5 text-15px leading-7 text-white/62">面向日常巡检和批量更新的界面，优先保证密度、可读性和状态感。</p>
      </div>
      <div class="relative grid grid-cols-3 gap-3 text-12px text-white/58">
        <div class="rounded-2 border border-white/10 bg-white/6 p-3">
          <div class="text-white font-700">Active</div>
          <div class="mt-1">24小时内更新</div>
        </div>
        <div class="rounded-2 border border-white/10 bg-white/6 p-3">
          <div class="text-white font-700">Stale</div>
          <div class="mt-1">超时待确认</div>
        </div>
        <div class="rounded-2 border border-white/10 bg-white/6 p-3">
          <div class="text-white font-700">Archive</div>
          <div class="mt-1">30天快照保留</div>
        </div>
      </div>
    </section>

    <section class="flex items-center justify-center bg-[#f6f8fb] px-8 text-ink-900">
      <div class="panel w-102 p-7">
        <div class="mb-7">
          <div class="text-22px font-800 text-ink-950">登录</div>
          <div class="mt-1 text-13px text-ink-500">使用你的 CMDB 账号继续</div>
        </div>

        <div v-if="error" class="mb-4 rounded-2 border border-red-200 bg-red-50 px-3 py-2 text-13px text-red-700">
          {{ error }}
        </div>

        <label class="block text-12px font-600 text-ink-500">用户名</label>
        <input v-model="username" class="field mt-1 w-full" placeholder="请输入用户名" @keyup.enter="doLogin" />

        <label class="mt-4 block text-12px font-600 text-ink-500">密码</label>
        <input v-model="password" class="field mt-1 w-full" type="password" placeholder="请输入密码" @keyup.enter="doLogin" />

        <button class="btn-primary mt-6 w-full" :disabled="loading" @click="doLogin">
          <span :class="loading ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-log-in'" />
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </div>
    </section>
  </main>
</template>
