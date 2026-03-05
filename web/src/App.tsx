import React, { useState, useEffect, useCallback, useMemo } from 'react'
import Reader from './components/Reader'
import ConfigForm from './components/ConfigForm'
import { api } from './api'
import type { SourceStats } from './types'
import './App.css'

function App() {
  const [view, setView] = useState('reader')
  const [sourceFilter, setSourceFilter] = useState('')
  const [feeds, setFeeds] = useState<SourceStats[]>([])
  const [feedQuery, setFeedQuery] = useState('')

  const loadFeeds = useCallback(async () => {
    try {
      const data = await api.getSourceStats(12)
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
    return feeds.filter(feed => feed.source.toLowerCase().includes(q))
  }, [feedQuery, feeds])

  return (
    <div className="app-container">
      <aside className="sidebar">
        <div className="sidebar-header">
          <h1>Sieve</h1>
        </div>
        <nav className="nav">
          <div 
            className={`nav-item ${view === 'reader' ? 'active' : ''}`}
            onClick={() => setView('reader')}
          >
            Reader
          </div>
          <div 
            className={`nav-item ${view === 'config' ? 'active' : ''}`}
            onClick={() => setView('config')}
          >
            Settings
          </div>
        </nav>
        {view === 'reader' && (
          <section className="sidebar-section">
            <h2>Feeds</h2>
            <input
              className="feed-search"
              type="text"
              placeholder="Search feeds"
              value={feedQuery}
              onChange={(e) => setFeedQuery(e.target.value)}
            />
            <button
              className={`feed-item ${sourceFilter === '' ? 'active' : ''}`}
              onClick={() => setSourceFilter('')}
            >
              <span className="feed-name">All Feeds</span>
            </button>
            {filteredFeeds.map((feed) => (
              <button
                key={feed.source}
                className={`feed-item ${sourceFilter === feed.source ? 'active' : ''}`}
                onClick={() => setSourceFilter(feed.source)}
              >
                <span className="feed-name">{feed.source}</span>
                <span className="feed-counts">{feed.high_interest}/{feed.saved}/{feed.visible}</span>
              </button>
            ))}
          </section>
        )}
      </aside>

      <main className="main-content">
        {view === 'reader' && <Reader sourceFilter={sourceFilter} onDataRefresh={loadFeeds} />}
        {view === 'config' && <ConfigForm />}
      </main>
    </div>
  )
}

export default App
