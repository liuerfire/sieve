import React, { useCallback, useEffect, useState } from 'react'
import ItemCard from './ItemCard'
import { api } from '../api'
import type { Item, AsyncState, ItemStats, SourceSuggestion } from '../types'

type ReaderMode = 'all' | 'saved' | 'digest'

interface ReaderProps {
  feedIDFilter: string
  onDataRefresh?: () => void
  refreshVersion?: number
}

const levelOptions = [
  { value: '', label: 'All levels' },
  { value: 'high_interest', label: 'High interest' },
  { value: 'interest', label: 'Interest' },
  { value: 'uninterested', label: 'Uninterested' },
  { value: 'exclude', label: 'Exclude' },
]

const formatDigestLabel = (days: number) => {
  switch (days) {
    case 14:
      return '14 days'
    case 30:
      return '30 days'
    default:
      return '7 days'
  }
}

const Reader: React.FC<ReaderProps> = ({ feedIDFilter, onDataRefresh, refreshVersion = 0 }) => {
  const [mode, setMode] = useState<ReaderMode>('all')
  const [qInput, setQInput] = useState('')
  const [q, setQ] = useState('')
  const [level, setLevel] = useState('')
  const [unreadOnly, setUnreadOnly] = useState(false)
  const [digestDays, setDigestDays] = useState(7)
  const [stats, setStats] = useState<ItemStats | null>(null)
  const [sourceSuggestions, setSourceSuggestions] = useState<SourceSuggestion[]>([])
  const [itemsState, setItemsState] = useState<AsyncState<Item[]>>({
    data: null,
    state: 'idle',
    error: null,
  })

  useEffect(() => {
    const timer = setTimeout(() => setQ(qInput.trim()), 300)
    return () => clearTimeout(timer)
  }, [qInput])

  const fetchItems = useCallback(async () => {
    setItemsState(prev => ({ ...prev, state: 'loading', error: null }))

    try {
      let data: Item[] = []
      if (mode === 'digest') {
        data = await api.getDigest(digestDays)
      } else if (mode === 'saved') {
        data = await api.searchItems({ q, feed_id: feedIDFilter, level, saved: true, unread: unreadOnly })
      } else if (q || feedIDFilter || level || unreadOnly) {
        data = await api.searchItems({ q, feed_id: feedIDFilter, level, unread: unreadOnly })
      } else {
        data = await api.getItems()
      }
      setItemsState({ data, state: 'success', error: null })
      const metrics = await api.getStats()
      setStats(metrics)
      const suggestions = await api.getSourceSuggestions(5, 3)
      setSourceSuggestions(suggestions)
      onDataRefresh?.()
    } catch (err) {
      setItemsState({
        data: null,
        state: 'error',
        error: err instanceof Error ? err.message : 'Failed to fetch items',
      })
    }
  }, [digestDays, level, mode, q, feedIDFilter, unreadOnly, onDataRefresh])

  useEffect(() => {
    fetchItems()
  }, [fetchItems, refreshVersion])

  const handleRefresh = useCallback(() => {
    fetchItems()
  }, [fetchItems])

  const handleUpdate = useCallback(() => {
    fetchItems()
  }, [fetchItems])

  const handleBulkRead = useCallback(async (read: boolean) => {
    if (!itemsState.data || itemsState.data.length === 0) return
    const ids = itemsState.data.map(item => item.ID)
    try {
      await api.bulkUpdateRead(ids, read)
      fetchItems()
    } catch {
      fetchItems()
    }
  }, [itemsState.data, fetchItems])

  return (
    <div className="reader-page">
      {itemsState.state === 'error' && (
        <div className="error-message" role="alert">
          <p>Error loading items: {itemsState.error}</p>
          <button onClick={handleRefresh} className="button button-primary">
            Try again
          </button>
        </div>
      )}

      {itemsState.state === 'loading' && !itemsState.data && (
        <div className="loading-state">
          <div className="spinner" aria-label="Loading items" />
          <p>Loading news items...</p>
        </div>
      )}

      {itemsState.state === 'success' && itemsState.data?.length === 0 && (
        <div className="empty-state card">
          <p>No items found. Run the aggregator to fetch news.</p>
          <button onClick={handleRefresh} className="button button-primary">
            Check again
          </button>
        </div>
      )}

      {itemsState.data && itemsState.data.length > 0 && (
        <div className="reader-layout">
          <aside className="reader-sidebar">
            {itemsState.data.some(item => item.InterestLevel === 'high_interest') && (
              <div className="highlights-card">
                <h3 className="highlights-title">⭐⭐ Highlights</h3>
                <ul className="highlights-list">
                  {itemsState.data
                    .filter(item => item.InterestLevel === 'high_interest')
                    .map(item => (
                      <li key={item.ID} className="highlight-item">
                        <span className="stars">⭐⭐</span>
                        <a href={`#item-${item.ID}`}>{item.Title}</a>
                      </li>
                    ))}
                </ul>
              </div>
            )}
          </aside>

          <main className="reader-main">
            <section className="story-grid" role="feed" aria-label="News items">
              {itemsState.data.map((item) => (
                <ItemCard key={item.ID} item={item} onUpdate={handleUpdate} />
              ))}
            </section>
          </main>
        </div>
      )}
    </div>
  )
}

export default Reader
