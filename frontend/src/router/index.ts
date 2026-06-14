import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
  },
  {
    path: '/',
    redirect: '/machine-info',
  },
  {
    path: '/machine-info',
    name: 'MachineInfo',
    component: () => import('../views/MachineInfo.vue'),
    meta: { title: '机器信息' },
  },
  {
    path: '/network-info',
    name: 'NetworkInfo',
    component: () => import('../views/NetworkInfo.vue'),
    meta: { title: '网络信息' },
  },
  {
    path: '/idc-info',
    name: 'IDCInfo',
    component: () => import('../views/IDCInfo.vue'),
    meta: { title: 'SSH信息' },
  },
  {
    path: '/business-info',
    name: 'BusinessInfo',
    component: () => import('../views/BusinessInfo.vue'),
    meta: { title: '业务信息' },
  },
  {
    path: '/deletion',
    name: 'Deletion',
    component: () => import('../views/Deletion.vue'),
    meta: { title: '删除管理' },
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (to.name !== 'Login' && !token) {
    next({ name: 'Login' })
  } else if (to.name === 'Login' && token) {
    next({ path: '/' })
  } else {
    next()
  }
})

export default router
