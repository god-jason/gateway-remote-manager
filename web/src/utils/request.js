import axios from 'axios'
import Cookies from 'js-cookie'
import { ElMessage } from 'element-plus'
import router from '@/router'

const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 30000
})

request.interceptors.request.use((config) => {
  const token = Cookies.get('token')
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`
  }
  return config
})

request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code === 401) {
      Cookies.remove('token')
      router.replace('/login')
      ElMessage.error(res.message || '登录已过期')
      return Promise.reject(new Error(res.message))
    }
    if (res.code !== 0) {
      ElMessage.error(res.message || '请求失败')
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return res
  },
  (error) => {
    const msg = error.response?.data?.message || error.message || '网络错误'
    if (error.response?.status === 401) {
      Cookies.remove('token')
      router.replace('/login')
    }
    ElMessage.error(msg)
    return Promise.reject(error)
  }
)

export default request