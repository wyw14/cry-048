import axios, { AxiosInstance, AxiosError } from 'axios'
import type { APIErrorEnvelope } from '@/types/models'

const baseURL = '/api/v1'

export const apiClient: AxiosInstance = axios.create({
  baseURL,
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

apiClient.interceptors.request.use((config) => {
  const user = localStorage.getItem('user_id') || 'user-local'
  config.headers['X-User-ID'] = user
  return config
})

apiClient.interceptors.response.use(
  (resp) => resp,
  (error: AxiosError<APIErrorEnvelope>) => {
    if (error.response) {
      const body = error.response.data?.error
      if (body) {
        const err = new Error(body.message) as Error & {
          code?: string
          requestId?: string
          fields?: { field: string; tag: string; value?: string; reason: string }[]
        }
        err.code = body.code
        err.requestId = body.request_id
        err.fields = body.field_errors
        return Promise.reject(err)
      }
    }
    return Promise.reject(error)
  },
)

export function unwrap<T>(promise: Promise<{ data: { data: T } }>): Promise<T> {
  return promise.then((r) => r.data.data)
}

export function unwrapList<T>(
  promise: Promise<{
    data: { data: T[]; total: number; page: number; page_size: number }
  }>,
): Promise<{ items: T[]; total: number; page: number; pageSize: number }> {
  return promise.then((r) => ({
    items: r.data.data,
    total: r.data.total,
    page: r.data.page,
    pageSize: r.data.page_size,
  }))
}
