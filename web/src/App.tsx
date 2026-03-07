import React, { startTransition, useState, useEffect, useCallback, useMemo, useRef } from 'react'
import Reader from './components/Reader'
import ConfigForm from './components/ConfigForm'
import { api } from './api'
import type { Feed, RefreshStatus } from './types'
import './App.css'

type View = 'reader' | 'config'

function App() {
  const [view, setView] = useState<View>('reader')
  const [feedFilter, setFeedFilter] = useState('')
  const [feeds, setFeeds] = useState<Feed[]>([])
  const [feedQuery, setFeedQuery] = useState('')
  const [refreshStatus, setRefreshStatus] = useState<RefreshStatus | null>(null)
  const [refreshMessage, setRefreshMessage] = useState('')
  const [refreshVersion, setRefreshVersion] = useState(0)
  const prevRefreshStatus = useRef<RefreshStatus | null>(null)

  const loadFeeds = useCallback(async () => {
    try {
      const data = await api.getFeeds(true)
      setFeeds(data)
    } catch {
      setFeeds([])
    }
  }, [])

  useEffect(() => {
    if (view === 'reader') {
      loadFeeds()
    }
  }, [view, loadFeeds])

  const filteredFeeds = useMemo(() => {
    const q = feedQuery.trim().toLowerCase()
    if (!q) return feeds
    return feeds.filter(feed => feed.name.toLowerCase().includes(q))
  }, [feedQuery, feeds])

  const loadRefreshStatus = useCallback(async () => {
    try {
      const status = await api.getRefreshStatus()
      const previous = prevRefreshStatus.current
      setRefreshStatus(status)
      prevRefreshStatus.current = status

      if (previous?.running && !status.running && status.last_completed_at !== previous.last_completed_at) {
        startTransition(() => {
          setRefreshVersion(v => v + 1)
        })
        loadFeeds()
      }
    } catch (err) {
      setRefreshMessage(err instanceof Error ? err.message : 'Failed to load refresh status')
    }
  }, [loadFeeds])

  useEffect(() => {
    loadRefreshStatus()
    const timer = window.setInterval(() => {
      loadRefreshStatus()
    }, 5000)
    return () => window.clearInterval(timer)
  }, [loadRefreshStatus])

  const handleRefreshTrigger = useCallback(async () => {
    try {
      setRefreshMessage('')
      const status = await api.triggerRefresh()
      setRefreshStatus(status)
      prevRefreshStatus.current = status
    } catch (err) {
      setRefreshMessage(err instanceof Error ? err.message : 'Failed to trigger refresh')
      loadRefreshStatus()
    }
  }, [loadRefreshStatus])

  const refreshLabel = useMemo(() => {
    if (!refreshStatus) return 'Refresh unavailable'
    if (refreshStatus.running) return 'Refresh in progress'
    if (refreshStatus.last_error) return 'Last refresh failed'
    if (refreshStatus.last_completed_at) return 'Last refresh complete'
    return 'Ready to refresh'
  }, [refreshStatus])

  const pageTitle = view === 'reader' ? 'Reader' : 'Settings'
  const pageDescription =
    view === 'reader'
      ? 'Review items, filter by source, and trigger refreshes.'
      : 'Manage feeds, rules, and AI defaults.'

  return (
    <div className="app-shell">
      <aside className="app-sidebar">
        <div className="app-sidebar-inner">
          <div className="brand-block">
            <span className="brand-mark">S</span>
            <div>
              <div className="brand-name">Sieve</div>
              <div className="brand-tagline">RSS triage workspace</div>
            </div>
          </div>

          <nav className="top-nav sidebar-nav" aria-label="Primary">
            <button
              className={`top-nav-item ${view === 'reader' ? 'active' : ''}`}
              onClick={() => setView('reader')}
            >
              Reader
            </button>
            <button
              className={`top-nav-item ${view === 'config' ? 'active' : ''}`}
              onClick={() => setView('config')}
            >
              Settings
            </button>
          </nav>

          <section className="refresh-panel" aria-label="Refresh controls">
            <div className="refresh-panel-copy">
              <div className="refresh-panel-title">Refresh pipeline</div>
              <div className="refresh-panel-status">{refreshLabel}</div>
            </div>
            <button
              className="button button-primary refresh-panel-button"
              onClick={handleRefreshTrigger}
              disabled={refreshStatus?.running}
            >
              {refreshStatus?.running ? 'Refreshing...' : 'Refresh now'}
            </button>
            {refreshStatus?.last_result && (
              <div className="refresh-panel-meta">
                {refreshStatus.last_result.items_processed} items · {refreshStatus.last_result.items_high_interest} high interest
              </div>
            )}
            {refreshMessage && <div className="error-message">{refreshMessage}</div>}
          </section>
        </div>
      </aside>

      <div className="app-main">
        <header className="app-header">
          <div className="app-header-main">
            <div className="page-title-block">
              <div className="page-title">{pageTitle}</div>
              <div className="header-summary">{pageDescription}</div>
            </div>

            {view === 'reader' && (
              <label className="header-search" aria-label="Search feeds">
                <span className="sr-only">Search feeds</span>
                <input
                  type="search"
                  placeholder="Filter feeds"
                  value={feedQuery}
                  onChange={(e) => setFeedQuery(e.target.value)}
                />
              </label>
            )}
          </div>

          {view === 'reader' && (
            <div className="feed-chip-row" aria-label="Feed filters">
              <button
                className={`feed-chip ${feedFilter === '' ? 'active' : ''}`}
                onClick={() => setFeedFilter('')}
              >
                All feeds
              </button>
              {filteredFeeds.map((feed) => (
                <button
                  key={feed.id}
                  className={`feed-chip ${feedFilter === feed.id ? 'active' : ''}`}
                  onClick={() => setFeedFilter(feed.id)}
                  title={feed.name}
                >
                  {feed.name}
                </button>
              ))}
            </div>
          )}
        </header>

        <main className="page-shell">
          {view === 'reader' && (
            <Reader
              feedIDFilter={feedFilter}
              onDataRefresh={loadFeeds}
              refreshVersion={refreshVersion}
            />
          )}
          {view === 'config' && <ConfigForm />}
        </main>
      </div>
    </div>
  )
}

export default App
