import React, { startTransition, useState, useEffect, useCallback, useMemo, useRef } from 'react'
import Reader from './components/Reader'
import ConfigForm from './components/ConfigForm'
import { api } from './api'
import type { Feed, RefreshStatus, RefreshStreamEvent } from './types'
import './App.css'

type View = 'reader' | 'refresh' | 'config'
type StreamState = 'connecting' | 'open' | 'error' | 'closed'

const MAX_REFRESH_EVENTS = 250

function App() {
  const [view, setView] = useState<View>('reader')
  const [feedFilter, setFeedFilter] = useState('')
  const [feeds, setFeeds] = useState<Feed[]>([])
  const [feedQuery, setFeedQuery] = useState('')
  const [refreshStatus, setRefreshStatus] = useState<RefreshStatus | null>(null)
  const [refreshMessage, setRefreshMessage] = useState('')
  const [refreshVersion, setRefreshVersion] = useState(0)
  const [refreshEvents, setRefreshEvents] = useState<RefreshStreamEvent[]>([])
  const [refreshRunID, setRefreshRunID] = useState('')
  const [streamState, setStreamState] = useState<StreamState>('connecting')
  const prevRefreshStatus = useRef<RefreshStatus | null>(null)
  const refreshLogRef = useRef<HTMLDivElement | null>(null)
  const shouldStickToBottomRef = useRef(true)

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

  useEffect(() => {
    const source = new EventSource(api.getRefreshStreamURL())

    setStreamState('connecting')
    source.onopen = () => {
      setStreamState('open')
    }
    source.onerror = () => {
      setStreamState('error')
    }
    source.onmessage = (event) => {
      let nextEvent: RefreshStreamEvent
      try {
        nextEvent = JSON.parse(event.data) as RefreshStreamEvent
      } catch {
        return
      }

      setRefreshRunID(current => {
        if (nextEvent.kind === 'refresh_started' && nextEvent.run_id !== current) {
          setRefreshEvents([nextEvent])
          return nextEvent.run_id
        }
        return current || nextEvent.run_id
      })

      setRefreshEvents(current => {
        if (nextEvent.kind === 'refresh_started') {
          if (current[0]?.run_id === nextEvent.run_id) {
            return current
          }
          return [nextEvent]
        }

        const next = [...current, nextEvent]
        return next.length > MAX_REFRESH_EVENTS ? next.slice(next.length - MAX_REFRESH_EVENTS) : next
      })
    }

    return () => {
      setStreamState('closed')
      source.close()
    }
  }, [])

  useEffect(() => {
    const log = refreshLogRef.current
    if (!log || !shouldStickToBottomRef.current) {
      return
    }
    log.scrollTop = log.scrollHeight
  }, [refreshEvents])

  const handleRefreshLogScroll = useCallback(() => {
    const log = refreshLogRef.current
    if (!log) {
      return
    }
    const distanceFromBottom = log.scrollHeight - log.scrollTop - log.clientHeight
    shouldStickToBottomRef.current = distanceFromBottom < 48
  }, [])

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

  const streamLabel = useMemo(() => {
    switch (streamState) {
      case 'open':
        return 'Live stream connected'
      case 'error':
        return 'Live stream reconnecting'
      case 'closed':
        return 'Live stream disconnected'
      default:
        return 'Live stream connecting'
    }
  }, [streamState])

  const pageTitle =
    view === 'reader' ? 'Reader' : view === 'refresh' ? 'Refresh' : 'Settings'
  const pageDescription =
    view === 'reader'
      ? 'Review items and filter by source.'
      : view === 'refresh'
        ? 'Run ingestion and inspect the live event stream.'
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
              className={`top-nav-item ${view === 'refresh' ? 'active' : ''}`}
              onClick={() => setView('refresh')}
            >
              Refresh
            </button>
            <button
              className={`top-nav-item ${view === 'config' ? 'active' : ''}`}
              onClick={() => setView('config')}
            >
              Settings
            </button>
          </nav>
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
          {view === 'refresh' && (
            <section className="refresh-page" aria-label="Refresh page">
              <section className="refresh-page-panel" aria-label="Refresh controls">
                <div className="refresh-page-header">
                  <div>
                    <div className="refresh-panel-title">Refresh pipeline</div>
                    <div className="refresh-panel-status">{refreshLabel}</div>
                  </div>
                  <button
                    className="button button-primary refresh-page-button"
                    onClick={handleRefreshTrigger}
                    disabled={refreshStatus?.running}
                  >
                    {refreshStatus?.running ? 'Refreshing...' : 'Refresh now'}
                  </button>
                </div>

                <div className="refresh-page-meta">
                  {refreshStatus?.last_result
                    ? `${refreshStatus.last_result.items_processed} items · ${refreshStatus.last_result.items_high_interest} high interest`
                    : 'No completed refresh yet.'}
                </div>
                {refreshMessage && <div className="error-message">{refreshMessage}</div>}
              </section>

              <section className="refresh-log-panel" aria-label="Live refresh log">
                <div className="refresh-log-header">
                  <div>
                    <div className="refresh-log-title">Live refresh log</div>
                    <div className="refresh-log-status">
                      {streamLabel}
                      {refreshRunID ? ` · ${refreshRunID}` : ''}
                    </div>
                  </div>
                  <div className="refresh-log-count">{refreshEvents.length} events</div>
                </div>

                <div
                  ref={refreshLogRef}
                  className="refresh-log-body"
                  onScroll={handleRefreshLogScroll}
                >
                  {refreshEvents.length === 0 ? (
                    <div className="refresh-log-empty">No live refresh events yet.</div>
                  ) : (
                    refreshEvents.map((event) => (
                      <div key={`${event.run_id}-${event.seq}`} className="refresh-log-row">
                        <div className="refresh-log-time">{formatRefreshEventTime(event.timestamp)}</div>
                        <div className="refresh-log-main">
                          <div className="refresh-log-summary">
                            <span className={`refresh-log-kind refresh-log-kind-${event.kind}`}>
                              {event.kind === 'progress' ? event.type ?? event.kind : event.kind}
                            </span>
                            {event.source && <span>{event.source}</span>}
                            {event.item && <span className="refresh-log-item">{event.item}</span>}
                            {event.level && <span className={`story-tag level-${event.level}`}>{event.level}</span>}
                          </div>
                          {(event.message || event.error || typeof event.count === 'number') && (
                            <div className="refresh-log-detail">
                              {event.message || event.error}
                              {typeof event.count === 'number' && typeof event.total === 'number'
                                ? ` (${event.count}/${event.total})`
                                : ''}
                            </div>
                          )}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </section>
            </section>
          )}
          {view === 'config' && <ConfigForm />}
        </main>
      </div>
    </div>
  )
}

function formatRefreshEventTime(timestamp: string): string {
  const value = new Date(timestamp)
  if (Number.isNaN(value.getTime())) {
    return '--:--:--'
  }
  return value.toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export default App
