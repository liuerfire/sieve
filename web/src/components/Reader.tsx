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
      <section className="news-hero">
        <div>
          <p className="eyebrow">Overview</p>
          <h1>Incoming items</h1>
          <p className="hero-copy">
            Review the latest stories, filter the stream, and update read state without leaving the queue.
          </p>
        </div>
        <button
          className="button button-primary"
          onClick={handleRefresh}
          disabled={itemsState.state === 'loading'}
          aria-label="Refresh news items"
        >
          {itemsState.state === 'loading' ? 'Refreshing...' : 'Refresh'}
        </button>
      </section>

      {stats && (
        <section className="stats-strip" aria-label="Reader stats">
          <div className="stat-card">
            <div className="stat-label">Visible</div>
            <div className="stat-value">{stats.total_visible}</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">Saved</div>
            <div className="stat-value">{stats.saved}</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">High Interest</div>
            <div className="stat-value">{stats.high_interest}</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">Unread</div>
            <div className="stat-value">{stats.unread_visible}</div>
          </div>
        </section>
      )}

      <section className="reader-toolbar card">
        <div className="chip-row" role="tablist" aria-label="Reader modes">
          <button className={`chip ${mode === 'all' ? 'active' : ''}`} onClick={() => setMode('all')}>
            All
          </button>
          <button className={`chip ${mode === 'saved' ? 'active' : ''}`} onClick={() => setMode('saved')}>
            Saved
          </button>
          <button className={`chip ${mode === 'digest' ? 'active' : ''}`} onClick={() => setMode('digest')}>
            Digest
          </button>
        </div>

        {mode !== 'digest' ? (
          <div className="toolbar-grid">
            <label className="search-field toolbar-search">
              <span className="sr-only">Search items</span>
              <input
                className="form-control"
                placeholder="Search keywords"
                value={qInput}
                onChange={e => setQInput(e.target.value)}
              />
            </label>

            <div className="chip-row compact" aria-label="Interest level filter">
              {levelOptions.map((option) => (
                <button
                  key={option.value || 'all'}
                  className={`chip ${level === option.value ? 'active' : ''}`}
                  onClick={() => setLevel(option.value)}
                >
                  {option.label}
                </button>
              ))}
            </div>

            <div className="utility-row">
              <label className={`chip toggle-chip ${unreadOnly ? 'active' : ''}`}>
                <input
                  type="checkbox"
                  checked={unreadOnly}
                  onChange={e => setUnreadOnly(e.target.checked)}
                />
                Unread only
              </label>
              <button className="button button-outline" onClick={() => handleBulkRead(true)}>
                Mark visible read
              </button>
              <button className="button button-outline" onClick={() => handleBulkRead(false)}>
                Mark visible unread
              </button>
            </div>
          </div>
        ) : (
          <div className="utility-row">
            <span className="toolbar-label">Digest range</span>
            {[7, 14, 30].map((days) => (
              <button
                key={days}
                className={`chip ${digestDays === days ? 'active' : ''}`}
                onClick={() => setDigestDays(days)}
              >
                {formatDigestLabel(days)}
              </button>
            ))}
          </div>
        )}
      </section>

      {sourceSuggestions.length > 0 && (
        <section className="briefing-card card">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Source analysis</p>
              <h2>Prune candidates</h2>
            </div>
          </div>
          <div className="source-stats-list">
            {sourceSuggestions.map((sg) => (
              <div key={sg.source} className="source-stats-row">
                <span className="source-name">{sg.source}</span>
                <span className="source-metrics">{sg.visible} visible · {sg.reason}</span>
              </div>
            ))}
          </div>
        </section>
      )}

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
        <section className="story-grid" role="feed" aria-label="News items">
          {itemsState.data.map((item) => (
            <ItemCard key={item.ID} item={item} onUpdate={handleUpdate} />
          ))}
        </section>
      )}
    </div>
  )
}

export default Reader
