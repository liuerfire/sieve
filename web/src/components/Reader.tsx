import React, { useEffect, useState, useCallback } from 'react'
import ItemCard from './ItemCard'
import { api } from '../api'
import type { Item, AsyncState, ItemStats, SourceSuggestion } from '../types'

type ReaderMode = 'all' | 'saved' | 'digest'

interface ReaderProps {
  feedIDFilter: string
  onDataRefresh?: () => void
}

const Reader: React.FC<ReaderProps> = ({ feedIDFilter, onDataRefresh }) => {
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
  }, [fetchItems])

  const handleRefresh = useCallback(() => {
    fetchItems()
  }, [fetchItems])

  const handleUpdate = useCallback(() => {
    // Refresh items after an update
    fetchItems()
  }, [fetchItems])

  const handleBulkRead = useCallback(async (read: boolean) => {
    if (!itemsState.data || itemsState.data.length === 0) return
    const ids = itemsState.data.map(item => item.ID)
    try {
      await api.bulkUpdateRead(ids, read)
      fetchItems()
    } catch {
      // Reuse existing error presentation by forcing a refresh path.
      fetchItems()
    }
  }, [itemsState.data, fetchItems])

  return (
    <div className="reader">
      <div className="reader-header">
        <h2>Recent News</h2>
        <button 
          className="button button-outline" 
          onClick={handleRefresh}
          disabled={itemsState.state === 'loading'}
          aria-label="Refresh news items"
        >
          {itemsState.state === 'loading' ? 'Loading...' : 'Refresh'}
        </button>
      </div>

      {stats && (
        <div className="stats-grid">
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
        </div>
      )}

      {sourceSuggestions.length > 0 && (
        <div className="card">
          <h3>Prune Candidates</h3>
          <div className="source-stats-list">
            {sourceSuggestions.map((sg) => (
              <div key={sg.source} className="source-stats-row">
                <span className="source-name">{sg.source}</span>
                <span className="source-metrics">{sg.visible} visible: {sg.reason}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="card">
        <div className="card-meta">
          <button className={`button-outline ${mode === 'all' ? 'active' : ''}`} onClick={() => setMode('all')}>
            All
          </button>
          <button className={`button-outline ${mode === 'saved' ? 'active' : ''}`} onClick={() => setMode('saved')}>
            Saved
          </button>
          <button className={`button-outline ${mode === 'digest' ? 'active' : ''}`} onClick={() => setMode('digest')}>
            Digest
          </button>
        </div>
        {mode !== 'digest' && (
          <div className="card-meta">
            <input
              className="form-control"
              placeholder="Search keywords"
              value={qInput}
              onChange={e => setQInput(e.target.value)}
            />
            <select className="form-control" value={level} onChange={e => setLevel(e.target.value)}>
              <option value="">All levels</option>
              <option value="high_interest">High Interest</option>
              <option value="interest">Interest</option>
              <option value="uninterested">Uninterested</option>
              <option value="exclude">Exclude</option>
            </select>
            <label>
              <input
                type="checkbox"
                checked={unreadOnly}
                onChange={e => setUnreadOnly(e.target.checked)}
              />
              {' '}Unread only
            </label>
            <button className="button-outline" onClick={() => handleBulkRead(true)}>
              Mark Visible Read
            </button>
            <button className="button-outline" onClick={() => handleBulkRead(false)}>
              Mark Visible Unread
            </button>
          </div>
        )}
        {mode === 'digest' && (
          <div className="card-meta">
            <label htmlFor="digest-days">Digest range:</label>
            <select
              id="digest-days"
              className="form-control"
              value={digestDays}
              onChange={e => setDigestDays(Number(e.target.value))}
            >
              <option value={7}>Last 7 days</option>
              <option value={14}>Last 14 days</option>
              <option value={30}>Last 30 days</option>
            </select>
          </div>
        )}
      </div>

      {itemsState.state === 'error' && (
        <div className="error-message" role="alert">
          <p>Error loading items: {itemsState.error}</p>
          <button onClick={handleRefresh} className="button button-primary">
            Try Again
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
        <div className="empty-state">
          <p>No items found. Run the aggregator to fetch news!</p>
          <button onClick={handleRefresh} className="button button-primary">
            Check Again
          </button>
        </div>
      )}

      {itemsState.data && itemsState.data.length > 0 && (
        <div className="items-list" role="feed" aria-label="News items">
          {itemsState.data.map((item) => (
            <ItemCard 
              key={item.ID} 
              item={item} 
              onUpdate={handleUpdate}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default Reader
