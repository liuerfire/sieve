import React, { useState, useEffect, useCallback, useMemo } from 'react'
import Reader from './components/Reader'
import ConfigForm from './components/ConfigForm'
import { api } from './api'
import type { Feed } from './types'
import './App.css'

type View = 'reader' | 'config'

function App() {
  const [view, setView] = useState<View>('reader')
  const [feedFilter, setFeedFilter] = useState('')
  const [feeds, setFeeds] = useState<Feed[]>([])
  const [feedQuery, setFeedQuery] = useState('')

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

  return (
    <div className="app-shell">
      <aside className="app-sidebar">
        <div className="app-sidebar-inner">
          <div className="brand-block">
            <span className="brand-mark">S</span>
            <div>
              <div className="brand-name">Sieve</div>
              <div className="brand-tagline">AI news briefings for your feeds</div>
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
        </div>
      </aside>

      <div className="app-main">
        <header className="app-header">
          <div className="app-header-main">
            {view === 'reader' ? (
              <label className="header-search" aria-label="Search feeds">
                <span className="sr-only">Search feeds</span>
                <input
                  type="search"
                  placeholder="Search feeds"
                  value={feedQuery}
                  onChange={(e) => setFeedQuery(e.target.value)}
                />
              </label>
            ) : (
              <div className="header-summary">Tune rules, AI preferences, and feed coverage.</div>
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
          {view === 'reader' && <Reader feedIDFilter={feedFilter} onDataRefresh={loadFeeds} />}
          {view === 'config' && <ConfigForm />}
        </main>
      </div>
    </div>
  )
}

export default App
