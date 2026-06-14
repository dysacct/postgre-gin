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
  <div class="login-card">
    <h1>CMDB</h1>
    <p class="desc">资产管理系统 · 请登录</p>

    <div v-if="error" class="error-msg">{{ error }}</div>

    <div class="form-group">
      <label>用户名</label>
      <input v-model="username" placeholder="请输入用户名" @keyup.enter="doLogin" />
    </div>
    <div class="form-group">
      <label>密码</label>
      <input v-model="password" type="password" placeholder="请输入密码" @keyup.enter="doLogin" />
    </div>

    <button class="login-btn" :disabled="loading" @click="doLogin">
      {{ loading ? '登录中...' : '登 录' }}
    </button>
  </div>
</template>
