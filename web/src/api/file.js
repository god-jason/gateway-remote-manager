import axios from 'axios'
import Cookies from 'js-cookie'
import request from '@/utils/request'

// 上传文件（原始字节流式上传到网关）
export function uploadFile(deviceId, file, path, overwrite = true, onProgress) {
  return request.post(`/device/${deviceId}/file/upload`, file, {
    params: { path, overwrite },
    headers: { 'Content-Type': 'application/octet-stream' },
    timeout: 0,
    onUploadProgress: (e) => {
      if (onProgress && e.total) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    }
  })
}

// 下载文件（原始 axios + blob，绕过 JSON 响应拦截器）
export async function downloadFile(deviceId, path, filename) {
  const res = await axios.get(`/api/device/${deviceId}/file/download`, {
    params: { path },
    responseType: 'blob',
    timeout: 0,
    headers: { Authorization: `Bearer ${Cookies.get('token')}` }
  })
  const url = URL.createObjectURL(res.data)
  const a = document.createElement('a')
  a.href = url
  a.download = filename || path.split('/').pop() || 'download'
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}