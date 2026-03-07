import type { Item, Feed, Settings, ItemStats, SourceStats, SourceSuggestion, RefreshStatus, CreateFeedInput } from './types'

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

  async getSources(): Promise<string[]> {
    return fetchWithErrorHandling<string[]>(`${API_BASE}/items/sources`)
  },

  async getFeeds(enabledOnly = false): Promise<Feed[]> {
    const qs = enabledOnly ? '?enabled=true' : ''
    return fetchWithErrorHandling<Feed[]>(`${API_BASE}/feeds${qs}`)
  },

  async createFeed(feed: CreateFeedInput): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/feeds`, {
      method: 'POST',
      body: JSON.stringify(feed),
    })
  },

  async updateFeed(id: string, feed: Partial<Feed>): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/feeds/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: JSON.stringify(feed),
    })
  },

  async deleteFeed(id: string): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/feeds/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  },

  async getSettings(): Promise<Settings> {
    return fetchWithErrorHandling<Settings>(`${API_BASE}/settings`)
  },

  async patchSettings(settings: Settings): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/settings`, {
      method: 'PATCH',
      body: JSON.stringify(settings),
    })
  },

  async getStats(): Promise<ItemStats> {
    return fetchWithErrorHandling<ItemStats>(`${API_BASE}/items/stats`)
  },

  async getSourceStats(limit = 10): Promise<SourceStats[]> {
    return fetchWithErrorHandling<SourceStats[]>(`${API_BASE}/items/source-stats?limit=${limit}`)
  },

  async getSourceSuggestions(minVisible = 10, limit = 10): Promise<SourceSuggestion[]> {
    return fetchWithErrorHandling<SourceSuggestion[]>(
      `${API_BASE}/items/source-suggestions?min_visible=${minVisible}&limit=${limit}`
    )
  },

  async updateItem(
    id: string,
    updates: {
      level?: string
      read?: boolean
      saved?: boolean
      user_interest_override?: string
    }
  ): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/items/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(updates),
    })
  },

  async bulkUpdateRead(ids: string[], read: boolean): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/items/bulk-read`, {
      method: 'POST',
      body: JSON.stringify({ ids, read }),
    })
  },

  async deleteItem(id: string): Promise<void> {
    return fetchWithErrorHandling<void>(`${API_BASE}/items/${id}`, {
      method: 'DELETE',
    })
  },

  async searchItems(params: {
    q?: string
    feed_id?: string
    source?: string
    level?: string
    saved?: boolean
    unread?: boolean
  }): Promise<Item[]> {
    const search = new URLSearchParams()
    if (params.q) search.set('q', params.q)
    if (params.feed_id) search.set('feed_id', params.feed_id)
    if (params.source) search.set('source', params.source)
    if (params.level) search.set('level', params.level)
    if (typeof params.saved === 'boolean') search.set('saved', String(params.saved))
    if (typeof params.unread === 'boolean') search.set('unread', String(params.unread))
    return fetchWithErrorHandling<Item[]>(`${API_BASE}/items/search?${search.toString()}`)
  },

  async getDigest(days = 7): Promise<Item[]> {
    return fetchWithErrorHandling<Item[]>(`${API_BASE}/digest?days=${days}`)
  },

  async getRefreshStatus(): Promise<RefreshStatus> {
    return fetchWithErrorHandling<RefreshStatus>(`${API_BASE}/refresh`)
  },

  async triggerRefresh(): Promise<RefreshStatus> {
    return fetchWithErrorHandling<RefreshStatus>(`${API_BASE}/refresh`, {
      method: 'POST',
    })
  },
}

export { ApiError }
