import request from '@/utils/request'

export const login = (data) => request.post('/login', data)
export const getMe = () => request.get('/me')