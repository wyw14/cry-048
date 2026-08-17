import { describe, expect, it } from 'vitest'
import { apiClient } from '@/api/client'

describe('apiClient', () => {
  it('exposes baseURL /api/v1', () => {
    expect(apiClient.defaults.baseURL).toBe('/api/v1')
  })
})
