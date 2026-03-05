import React, { useState, useEffect, useCallback, useMemo } from 'react'
import Reader from './components/Reader'
import ConfigForm from './components/ConfigForm'
import { api } from './api'
import type { Feed } from './types'
import './App.css'

function App() {
  const [view, setView] = useState('reader')
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
              className={`feed-item ${feedFilter === '' ? 'active' : ''}`}
              onClick={() => setFeedFilter('')}
            >
              <span className="feed-name">All Feeds</span>
            </button>
            {filteredFeeds.map((feed) => (
              <button
                key={feed.id}
                className={`feed-item ${feedFilter === feed.id ? 'active' : ''}`}
                onClick={() => setFeedFilter(feed.id)}
              >
                <span className="feed-name">{feed.name}</span>
              </button>
            ))}
          </section>
        )}
      </aside>

      <main className="main-content">
        {view === 'reader' && <Reader feedIDFilter={feedFilter} onDataRefresh={loadFeeds} />}
        {view === 'config' && <ConfigForm />}
      </main>
    </div>
  )
}

export default App
