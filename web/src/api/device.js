import request from '@/utils/request'

export const searchDevices = (params) => request.post('/device/search', params)
export const getDevice = (id) => request.get(`/device/${id}`)
export const deleteDevice = (id) => request.delete(`/device/${id}`)
export const dashboardSummary = () => request.get('/dashboard/summary')