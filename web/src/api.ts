import type { Item, Config } from './types'

const API_BASE = '/api'

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const text = await response.text()
    throw new ApiError(response.status, text || `HTTP ${response.status}`)
  }
  if (response.status === 204) {
    return undefined as T
  }
  return response.json()
}

async function fetchWithErrorHandling<T>(
  url: string,
  options?: RequestInit
): Promise<T> {
  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    })
    return handleResponse<T>(response)
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw new ApiError(0, error instanceof Error ? error.message : 'Network error')
  }
}

export const api = {
  // Items
  async getItems(): Promise<Item[]> {
    return fetchWithErrorHandling<Item[]>(`${API_BASE}/items`)
  },

  async updateItem(id: string, updates: { level?: string; read?: boolean }): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/items/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(updates),
    })
  },

  async deleteItem(id: string): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/items/${id}`, {
      method: 'DELETE',
    })
  },

  // Config
  async getConfig(): Promise<Config> {
    return fetchWithErrorHandling<Config>(`${API_BASE}/config`)
  },

  async saveConfig(config: Config): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/config`, {
      method: 'PUT',
      body: JSON.stringify(config),
    })
  },
}

export { ApiError }
