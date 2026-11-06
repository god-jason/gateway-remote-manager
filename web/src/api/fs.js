import request from '@/utils/request'

export const fsList = (deviceId, path) => request.post(`/device/${deviceId}/fs/list`, { path })
export const fsMkdir = (deviceId, path) => request.post(`/device/${deviceId}/fs/mkdir`, { path })
export const fsDelete = (deviceId, path) => request.post(`/device/${deviceId}/fs/delete`, { path })
export const fsRename = (deviceId, path, newPath) => request.post(`/device/${deviceId}/fs/rename`, { path, new_path: newPath })
export const fsInfo = (deviceId, path) => request.post(`/device/${deviceId}/fs/info`, { path })
export const fsRead = (deviceId, path) => request.post(`/device/${deviceId}/fs/read`, { path })
export const fsWrite = (deviceId, path, content) => request.post(`/device/${deviceId}/fs/write`, { path, content })