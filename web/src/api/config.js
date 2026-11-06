import request from '@/utils/request'

export const configList = (deviceId) => request.post(`/device/${deviceId}/config/list`, {})
export const configGet = (deviceId, configKey) => request.get(`/device/${deviceId}/config/get`, { params: { config_key: configKey } })
export const configSet = (deviceId, configKey, configType, content) =>
  request.post(`/device/${deviceId}/config/set`, { config_key: configKey, config_type: configType, content })
export const configDelete = (deviceId, configKey) => request.post(`/device/${deviceId}/config/delete`, { config_key: configKey })