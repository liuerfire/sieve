import React, { useState, useEffect } from 'react'

interface Source {
  name: string
  title: string
  url: string
}

interface Config {
  global: {
    high_interest: string
    interest: string
    uninterested: string
    exclude: string
    preferred_language: string
    ai?: { provider: string; model: string }
  }
  sources: Source[]
}

const ConfigForm: React.FC = () => {
  const [config, setConfig] = useState<Config | null>(null)
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState('')

  useEffect(() => {
    fetch('/api/config')
      .then(res => res.json())
      .then(data => {
        setConfig(data)
        setLoading(false)
      })
  }, [])

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault()
    setMessage('Saving...')
    fetch('/api/config', {
      method: 'PUT',
      body: JSON.stringify(config)
    }).then(() => setMessage('Config saved successfully!'))
  }

  const addSource = () => {
    if (!config) return
    setConfig({
      ...config,
      sources: [...config.sources, { name: 'new_source', title: 'New Source', url: '' }]
    })
  }

  const removeSource = (index: number) => {
    if (!config) return
    const sources = [...config.sources]
    sources.splice(index, 1)
    setConfig({ ...config, sources })
  }

  const updateSource = (index: number, field: string, value: string) => {
    if (!config) return
    const sources = [...config.sources]
    sources[index] = { ...sources[index], [field]: value }
    setConfig({ ...config, sources })
  }

  if (loading || !config) return <div>Loading config...</div>

  return (
    <div className="config-form">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '32px' }}>
        <h2 style={{ margin: 0 }}>Configuration</h2>
        <button className="button button-primary" onClick={handleSave}>Save Changes</button>
      </div>

      {message && <div style={{ background: '#dcfce7', color: '#166534', padding: '12px', borderRadius: '6px', marginBottom: '24px' }}>{message}</div>}

      <form onSubmit={handleSave}>
        <section className="card">
          <h3>Global Interest Rules</h3>
          <div className="form-group">
            <label>High Interest Keywords (⭐⭐)</label>
            <textarea 
              className="form-control" 
              value={config.global.high_interest} 
              onChange={e => setConfig({ ...config, global: { ...config.global, high_interest: e.target.value } })}
              rows={3}
            />
          </div>
          <div className="form-group">
            <label>Interest Keywords (⭐)</label>
            <textarea 
              className="form-control" 
              value={config.global.interest} 
              onChange={e => setConfig({ ...config, global: { ...config.global, interest: e.target.value } })}
              rows={3}
            />
          </div>
        </section>

        <section className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
            <h3 style={{ margin: 0 }}>RSS Sources</h3>
            <button type="button" className="button button-outline" onClick={addSource}>+ Add Source</button>
          </div>
          {config.sources.map((src, idx) => (
            <div key={idx} className="source-item" style={{ display: 'flex', gap: '16px', marginBottom: '16px', borderBottom: '1px solid #eee', paddingBottom: '16px' }}>
              <div style={{ flex: 1 }}>
                <input 
                  className="form-control" 
                  value={src.name} 
                  placeholder="name"
                  onChange={e => updateSource(idx, 'name', e.target.value)}
                />
              </div>
              <div style={{ flex: 2 }}>
                <input 
                  className="form-control" 
                  value={src.url} 
                  placeholder="URL"
                  onChange={e => updateSource(idx, 'url', e.target.value)}
                />
              </div>
              <button type="button" className="button" style={{ color: '#ef4444' }} onClick={() => removeSource(idx)}>Remove</button>
            </div>
          ))}
        </section>

        <section className="card">
          <h3>AI Settings</h3>
          <div className="form-group">
            <label>Preferred Language</label>
            <input 
              className="form-control" 
              value={config.global.preferred_language} 
              onChange={e => setConfig({ ...config, global: { ...config.global, preferred_language: e.target.value } })}
            />
          </div>
          <div className="form-group">
            <label>AI Provider</label>
            <select 
              className="form-control" 
              value={config.global.ai?.provider} 
              onChange={e => setConfig({ ...config, global: { ...config.global, ai: { ...config.global.ai!, provider: e.target.value } } })}
            >
              <option value="gemini">Gemini</option>
              <option value="qwen">Qwen</option>
            </select>
          </div>
        </section>
      </form>
    </div>
  )
}

export default ConfigForm
