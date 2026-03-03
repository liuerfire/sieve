import type { Item, Config, ProgressEvent } from './types'

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

  // Run
  async startRun(): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/run`, {
      method: 'POST',
    })
  },
}

// EventSource with reconnection logic
interface EventSourceOptions {
  onMessage: (event: ProgressEvent) => void
  onError?: (error: Event) => void
  onOpen?: () => void
  maxRetries?: number
  retryDelay?: number
}

export function createEventSource(options: EventSourceOptions): () => void {
  const { onMessage, onError, onOpen, maxRetries = 5, retryDelay = 3000 } = options
  
  let eventSource: EventSource | null = null
  let retryCount = 0
  let retryTimeout: ReturnType<typeof setTimeout> | null = null
  let isClosed = false

  const connect = () => {
    if (isClosed) return

    eventSource = new EventSource(`${API_BASE}/events`)

    eventSource.onopen = () => {
      retryCount = 0
      onOpen?.()
    }

    eventSource.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data) as ProgressEvent
        onMessage(event)
      } catch (err) {
        console.error('Failed to parse event:', err)
      }
    }

    eventSource.onerror = (error) => {
      onError?.(error)
      
      if (eventSource?.readyState === EventSource.CLOSED) {
        eventSource.close()
        
        if (retryCount < maxRetries && !isClosed) {
          retryCount++
          retryTimeout = setTimeout(connect, retryDelay * retryCount)
        }
      }
    }
  }

  connect()

  return () => {
    isClosed = true
    if (retryTimeout) {
      clearTimeout(retryTimeout)
    }
    eventSource?.close()
  }
}

export { ApiError }
