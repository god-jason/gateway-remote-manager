import request from '@/utils/request'

export const taskCreate = (deviceId, command, timeoutSeconds = 60, taskType = 'command') =>
  request.post(`/device/${deviceId}/task/create`, {
    task_type: taskType,
    command,
    timeout_seconds: timeoutSeconds
  })
export const taskList = (deviceId) => request.post(`/device/${deviceId}/task/list`, {})
export const taskGet = (deviceId, taskId) => request.get(`/device/${deviceId}/task/get`, { params: { task_id: taskId } })
export const taskCancel = (deviceId, taskId) => request.post(`/device/${deviceId}/task/cancel`, { task_id: taskId })