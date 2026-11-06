import request from '@/utils/request'

export const releaseSearch = () => request.post('/upgrade/release/search', {})
export const releaseUpload = (formData) => request.post('/upgrade/release/upload', formData)
export const releaseDelete = (id) => request.delete(`/upgrade/release/${id}`)
export const releasePublish = (id) => request.put(`/upgrade/release/${id}/publish`)
export const releaseDisable = (id) => request.put(`/upgrade/release/${id}/disable`)
export const upgradeTaskSearch = () => request.post('/upgrade/task/search', {})
export const upgradeDevice = (deviceId, releaseId) => request.post(`/device/${deviceId}/upgrade`, { release_id: releaseId })