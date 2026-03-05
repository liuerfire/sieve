import React, { useEffect, useMemo, useState } from 'react'
import { api } from '../api'
import type { Feed, Settings } from '../types'

type SettingsDraft = Settings

const defaultFeed: Feed = {
  id: '',
  name: '',
  url: '',
  enabled: true,
  summarize: false,
}

const ConfigForm: React.FC = () => {
  const [feeds, setFeeds] = useState<Feed[]>([])
  const [settings, setSettings] = useState<SettingsDraft>({})
  const [newFeed, setNewFeed] = useState<Feed>(defaultFeed)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const [feedData, settingsData] = await Promise.all([
        api.getFeeds(false),
        api.getSettings(),
      ])
      setFeeds(feedData)
      setSettings(settingsData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  const canCreateFeed = useMemo(() => {
    return newFeed.id.trim() !== '' && newFeed.name.trim() !== '' && newFeed.url.trim() !== ''
  }, [newFeed])

  const handleCreateFeed = async () => {
    if (!canCreateFeed) return
    setSaving(true)
    setError(null)
    try {
      await api.createFeed({
        ...newFeed,
        id: newFeed.id.trim(),
        name: newFeed.name.trim(),
        url: newFeed.url.trim(),
      })
      setNewFeed(defaultFeed)
      await loadData()
      setMessage('Feed created')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create feed')
    } finally {
      setSaving(false)
    }
  }

  const handleUpdateFeed = async (feed: Feed) => {
    setSaving(true)
    setError(null)
    try {
      await api.updateFeed(feed.id, feed)
      setMessage('Feed updated')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update feed')
    } finally {
      setSaving(false)
    }
  }

  const handleDeleteFeed = async (id: string) => {
    setSaving(true)
    setError(null)
    try {
      await api.deleteFeed(id)
      setFeeds(prev => prev.filter(feed => feed.id !== id))
      setMessage('Feed deleted')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete feed')
    } finally {
      setSaving(false)
    }
  }

  const handleSaveSettings = async () => {
    setSaving(true)
    setError(null)
    try {
      await api.patchSettings(settings)
      setMessage('Settings saved')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="config-form loading">
        <div className="spinner" aria-label="Loading settings" />
        <p>Loading settings...</p>
      </div>
    )
  }

  return (
    <div className="config-form">
      <div className="form-header">
        <h2>Settings</h2>
        <button className="button button-primary" onClick={handleSaveSettings} disabled={saving}>
          {saving ? 'Saving...' : 'Save Settings'}
        </button>
      </div>

      {message && <div className="success-message">{message}</div>}
      {error && <div className="error-message">{error}</div>}

      <section className="card">
        <h3>Global Rules</h3>
        <div className="form-group">
          <label htmlFor="high-interest">High Interest</label>
          <textarea
            id="high-interest"
            className="form-control"
            rows={2}
            value={settings.high_interest || ''}
            onChange={(e) => setSettings(prev => ({ ...prev, high_interest: e.target.value }))}
          />
        </div>
        <div className="form-group">
          <label htmlFor="interest">Interest</label>
          <textarea
            id="interest"
            className="form-control"
            rows={2}
            value={settings.interest || ''}
            onChange={(e) => setSettings(prev => ({ ...prev, interest: e.target.value }))}
          />
        </div>
        <div className="form-group">
          <label htmlFor="uninterested">Uninterested</label>
          <textarea
            id="uninterested"
            className="form-control"
            rows={2}
            value={settings.uninterested || ''}
            onChange={(e) => setSettings(prev => ({ ...prev, uninterested: e.target.value }))}
          />
        </div>
        <div className="form-group">
          <label htmlFor="exclude">Exclude</label>
          <textarea
            id="exclude"
            className="form-control"
            rows={2}
            value={settings.exclude || ''}
            onChange={(e) => setSettings(prev => ({ ...prev, exclude: e.target.value }))}
          />
        </div>
      </section>

      <section className="card">
        <h3>AI</h3>
        <div className="form-group">
          <label htmlFor="preferred-language">Preferred Language</label>
          <input
            id="preferred-language"
            className="form-control"
            value={settings.preferred_language || ''}
            onChange={(e) => setSettings(prev => ({ ...prev, preferred_language: e.target.value }))}
          />
        </div>
        <div className="form-group">
          <label htmlFor="ai-provider">Provider</label>
          <select
            id="ai-provider"
            className="form-control"
            value={settings.ai_provider || ''}
            onChange={(e) => setSettings(prev => ({ ...prev, ai_provider: e.target.value }))}
          >
            <option value="">Default</option>
            <option value="gemini">Gemini</option>
            <option value="qwen">Qwen</option>
          </select>
        </div>
        <div className="form-group">
          <label htmlFor="ai-model">Model</label>
          <input
            id="ai-model"
            className="form-control"
            value={settings.ai_model || ''}
            onChange={(e) => setSettings(prev => ({ ...prev, ai_model: e.target.value }))}
          />
        </div>
      </section>

      <section className="card">
        <h3>Feeds</h3>
        <div className="source-list">
          {feeds.map((feed) => (
            <div className="source-item" key={feed.id}>
              <div className="source-header">
                <strong>{feed.name}</strong>
                <button className="button button-danger" onClick={() => handleDeleteFeed(feed.id)} disabled={saving}>
                  Delete
                </button>
              </div>
              <div className="form-group">
                <label>ID</label>
                <input className="form-control" value={feed.id} disabled />
              </div>
              <div className="form-group">
                <label>Name</label>
                <input
                  className="form-control"
                  value={feed.name}
                  onChange={(e) => setFeeds(prev => prev.map(f => f.id === feed.id ? { ...f, name: e.target.value } : f))}
                />
              </div>
              <div className="form-group">
                <label>URL</label>
                <input
                  className="form-control"
                  value={feed.url}
                  onChange={(e) => setFeeds(prev => prev.map(f => f.id === feed.id ? { ...f, url: e.target.value } : f))}
                />
              </div>
              <div className="form-group">
                <label>
                  <input
                    type="checkbox"
                    checked={feed.enabled}
                    onChange={(e) => setFeeds(prev => prev.map(f => f.id === feed.id ? { ...f, enabled: e.target.checked } : f))}
                  />{' '}
                  Enabled
                </label>
              </div>
              <button className="button button-outline" onClick={() => handleUpdateFeed(feed)} disabled={saving}>
                Update Feed
              </button>
            </div>
          ))}
        </div>

        <div className="source-item" style={{ marginTop: 16 }}>
          <h4>Add Feed</h4>
          <div className="form-group">
            <label>ID</label>
            <input
              className="form-control"
              value={newFeed.id}
              onChange={(e) => setNewFeed(prev => ({ ...prev, id: e.target.value }))}
            />
          </div>
          <div className="form-group">
            <label>Name</label>
            <input
              className="form-control"
              value={newFeed.name}
              onChange={(e) => setNewFeed(prev => ({ ...prev, name: e.target.value }))}
            />
          </div>
          <div className="form-group">
            <label>URL</label>
            <input
              className="form-control"
              value={newFeed.url}
              onChange={(e) => setNewFeed(prev => ({ ...prev, url: e.target.value }))}
            />
          </div>
          <button className="button button-primary" onClick={handleCreateFeed} disabled={!canCreateFeed || saving}>
            Add Feed
          </button>
        </div>
      </section>
    </div>
  )
}

export default ConfigForm
