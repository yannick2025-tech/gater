import axios from 'axios'
import type { AxiosInstance, AxiosError } from 'axios'
import { ElMessage } from 'element-plus'

const instance: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

instance.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code === 200) {
      return res.data
    }
    // 业务错误（code非200但HTTP状态码正常）
    const msg = res.message || '请求失败'
    ElMessage.error(msg)
    return Promise.reject(new Error(msg))
  },
  (error: AxiosError<{ code?: number; message?: string }>) => {
    // 优先取后端返回的业务message
    const serverMsg = error.response?.data?.message
    const httpStatus = error.response?.status

    let displayMsg = '网络错误，请稍后重试'

    if (serverMsg) {
      displayMsg = serverMsg
    } else if (error.code === 'ECONNABORTED') {
      displayMsg = '请求超时，请检查网络连接'
    } else if (!error.response) {
      displayMsg = '网络连接失败，请检查后端服务是否启动'
    }

    // 针对常见状态码的友好提示
    const statusTips: Record<number, string> = {
      401: '登录已过期，请重新登录',
      403: '没有权限执行此操作',
      404: '请求的资源不存在',
      500: '服务器内部错误，请联系管理员',
    }
    if (httpStatus && !serverMsg && statusTips[httpStatus]) {
      displayMsg = statusTips[httpStatus]
    }

    ElMessage.error(displayMsg)
    return Promise.reject(new Error(displayMsg))
  },
)

export default instance
