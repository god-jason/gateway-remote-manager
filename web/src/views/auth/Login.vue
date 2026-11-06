<template>
  <div class="login-wrap">
    <el-card class="login-card">
      <h2 class="title">Gateway RMM</h2>
      <p class="subtitle">网关远程运维管理平台</p>
      <el-form @submit.prevent="onSubmit">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" :prefix-icon="'User'" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" :prefix-icon="'Lock'" show-password />
        </el-form-item>
        <el-button type="primary" native-type="submit" class="submit" :loading="loading">
          登录
        </el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import Cookies from 'js-cookie'
import { login } from '@/api/auth'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()
const loading = ref(false)
const form = ref({ username: 'admin', password: '' })

async function onSubmit() {
  loading.value = true
  try {
    const res = await login(form.value)
    Cookies.set('token', res.data.token)
    userStore.setUsername(res.data.username || form.value.username)
    ElMessage.success('登录成功')
    router.replace('/dashboard')
  } catch (e) {
    // 错误已由拦截器提示
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f2b48 0%, #263b62 100%);
}
.login-card {
  width: 360px;
  padding: 12px 8px;
}
.title {
  text-align: center;
  margin: 0 0 4px;
}
.subtitle {
  text-align: center;
  color: #909399;
  font-size: 13px;
  margin: 0 0 24px;
}
.submit {
  width: 100%;
}
</style>