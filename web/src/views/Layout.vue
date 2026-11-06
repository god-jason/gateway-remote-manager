<template>
  <el-container class="layout">
    <el-aside width="200px" class="aside">
      <div class="logo">Gateway RMM</div>
      <el-menu
        :default-active="$route.path"
        router
        background-color="#001529"
        text-color="#a6adb4"
        active-text-color="#409EFF"
      >
        <el-menu-item index="/dashboard">
          <el-icon><Odometer /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>
        <el-menu-item index="/device">
          <el-icon><Monitor /></el-icon>
          <span>网关管理</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span>网关远程运维管理平台</span>
        <div class="header-right">
          <span class="username">{{ userStore.username || 'admin' }}</span>
          <el-button link type="danger" @click="logout">退出</el-button>
        </div>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { useRouter } from 'vue-router'
import Cookies from 'js-cookie'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()

function logout() {
  Cookies.remove('token')
  router.replace('/login')
}
</script>

<style scoped>
.layout {
  height: 100%;
}
.aside {
  background: #001529;
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  color: #fff;
  font-weight: 700;
  font-size: 16px;
}
.aside :deep(.el-menu) {
  border-right: none;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #e5e6eb;
  background: #fff;
}
.header-right {
  display: flex;
  align-items: center;
}
.username {
  margin-right: 12px;
  color: #606266;
}
.main {
  background: #f5f6f8;
}
</style>