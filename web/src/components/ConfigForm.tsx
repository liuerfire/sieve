import React, { useState, useEffect, useCallback } from 'react'
import { api } from '../api'
import type { Config, Source, AsyncState } from '../types'

interface ValidationError {
  field: string
  message: string
}

const ConfigForm: React.FC = () => {
  const [config, setConfig] = useState<Config | null>(null)
  const [originalConfig, setOriginalConfig] = useState<Config | null>(null)
  const [loadState, setLoadState] = useState<AsyncState<void>>({
    data: null,
    state: 'loading',
    error: null,
  })
  const [saveState, setSaveState] = useState<AsyncState<void>>({
    data: null,
    state: 'idle',
    error: null,
  })
  const [validationErrors, setValidationErrors] = useState<ValidationError[]>([])
  const [hasChanges, setHasChanges] = useState(false)

  useEffect(() => {
    loadConfig()
  }, [])

  useEffect(() => {
    if (config && originalConfig) {
      setHasChanges(JSON.stringify(config) !== JSON.stringify(originalConfig))
    }
  }, [config, originalConfig])

  const loadConfig = async () => {
    setLoadState({ data: null, state: 'loading', error: null })
    try {
      const data = await api.getConfig()
      setConfig(data)
      setOriginalConfig(JSON.parse(JSON.stringify(data)))
      setLoadState({ data: null, state: 'success', error: null })
    } catch (err) {
      setLoadState({
        data: null,
        state: 'error',
        error: err instanceof Error ? err.message : 'Failed to load config',
      })
    }
  }

  const validate = useCallback((): boolean => {
    if (!config) return false
    
    const errors: ValidationError[] = []

    // Validate sources
    config.sources.forEach((src, idx) => {
      if (!src.name.trim()) {
        errors.push({ field: `source-${idx}-name`, message: 'Source name is required' })
      }
      if (!src.url.trim()) {
        errors.push({ field: `source-${idx}-url`, message: 'Source URL is required' })
      } else {
        try {
          new URL(src.url)
        } catch {
          errors.push({ field: `source-${idx}-url`, message: 'Invalid URL format' })
        }
      }
    })

    // Validate AI settings
    if (config.global.ai?.provider && !['gemini', 'qwen'].includes(config.global.ai.provider)) {
      errors.push({ field: 'ai-provider', message: 'Invalid AI provider' })
    }

    setValidationErrors(errors)
    return errors.length === 0
  }, [config])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!validate() || !config) return

    setSaveState({ data: null, state: 'loading', error: null })
    
    try {
      await api.saveConfig(config)
      setOriginalConfig(JSON.parse(JSON.stringify(config)))
      setSaveState({ data: null, state: 'success', error: null })
      setTimeout(() => setSaveState(prev => ({ ...prev, state: 'idle' })), 3000)
    } catch (err) {
      setSaveState({
        data: null,
        state: 'error',
        error: err instanceof Error ? err.message : 'Failed to save config',
      })
    }
  }

  const addSource = () => {
    if (!config) return
    
    const newSource: Source = { 
      name: '', 
      title: '', 
      url: '' 
    }
    
    setConfig({
      ...config,
      sources: [...config.sources, newSource]
    })
  }

  const removeSource = (index: number) => {
    if (!config) return
    
    const sources = [...config.sources]
    sources.splice(index, 1)
    setConfig({ ...config, sources })
  }

  const updateSource = (index: number, field: keyof Source, value: string) => {
    if (!config) return
    
    const sources = [...config.sources]
    sources[index] = { ...sources[index], [field]: value }
    setConfig({ ...config, sources })
  }

  const getFieldError = (field: string): string | undefined => {
    return validationErrors.find(e => e.field === field)?.message
  }

  if (loadState.state === 'loading') {
    return (
      <div className="config-form loading">
        <div className="spinner" aria-label="Loading configuration" />
        <p>Loading configuration...</p>
      </div>
    )
  }

  if (loadState.state === 'error') {
    return (
      <div className="config-form error">
        <div className="error-message" role="alert">
          <p>Error loading configuration: {loadState.error}</p>
          <button onClick={loadConfig} className="button button-primary">
            Try Again
          </button>
        </div>
      </div>
    )
  }

  if (!config) return null

  return (
    <div className="config-form">
      <div className="form-header">
        <h2>Configuration</h2>
        <div className="form-actions">
          {hasChanges && (
            <span className="unsaved-indicator" aria-live="polite">
              Unsaved changes
            </span>
          )}
          <button 
            className="button button-primary" 
            onClick={handleSave}
            disabled={saveState.state === 'loading' || !hasChanges}
            aria-label="Save configuration changes"
          >
            {saveState.state === 'loading' ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>

      {saveState.state === 'success' && (
        <div className="success-message" role="status">
          Configuration saved successfully!
        </div>
      )}

      {saveState.state === 'error' && (
        <div className="error-message" role="alert">
          Failed to save: {saveState.error}
        </div>
      )}

      <form onSubmit={handleSave}>
        <section className="card">
          <h3>Global Interest Rules</h3>
          
          <div className="form-group">
            <label htmlFor="high-interest">
              High Interest Keywords <span aria-label="two stars">⭐⭐</span>
            </label>
            <textarea
              id="high-interest"
              className="form-control"
              value={config.global.high_interest}
              onChange={e => setConfig({ 
                ...config, 
                global: { ...config.global, high_interest: e.target.value } 
              })}
              rows={3}
              aria-describedby="high-interest-help"
            />
            <span id="high-interest-help" className="help-text">
              Comma-separated keywords for high-interest items
            </span>
          </div>

          <div className="form-group">
            <label htmlFor="interest">
              Interest Keywords <span aria-label="one star">⭐</span>
            </label>
            <textarea
              id="interest"
              className="form-control"
              value={config.global.interest}
              onChange={e => setConfig({ 
                ...config, 
                global: { ...config.global, interest: e.target.value } 
              })}
              rows={3}
              aria-describedby="interest-help"
            />
            <span id="interest-help" className="help-text">
              Comma-separated keywords for general interest items
            </span>
          </div>

          <div className="form-group">
            <label htmlFor="uninterested">Uninterested Keywords</label>
            <textarea
              id="uninterested"
              className="form-control"
              value={config.global.uninterested}
              onChange={e => setConfig({ 
                ...config, 
                global: { ...config.global, uninterested: e.target.value } 
              })}
              rows={2}
            />
          </div>

          <div className="form-group">
            <label htmlFor="exclude">Exclude Keywords</label>
            <textarea
              id="exclude"
              className="form-control"
              value={config.global.exclude}
              onChange={e => setConfig({ 
                ...config, 
                global: { ...config.global, exclude: e.target.value } 
              })}
              rows={2}
            />
          </div>
        </section>

        <section className="card">
          <div className="section-header">
            <h3>RSS Sources</h3>
            <button 
              type="button" 
              className="button button-outline" 
              onClick={addSource}
              aria-label="Add new RSS source"
            >
              + Add Source
            </button>
          </div>
          
          {config.sources.length === 0 && (
            <p className="empty-sources">No sources configured. Add one to get started.</p>
          )}
          
          {config.sources.map((src, idx) => (
            <div 
              key={`${src.name}-${idx}`} 
              className={`source-item ${getFieldError(`source-${idx}-name`) || getFieldError(`source-${idx}-url`) ? 'has-error' : ''}`}
            >
              <div className="source-field">
                <label htmlFor={`source-${idx}-name`} className="sr-only">Source Name</label>
                <input
                  id={`source-${idx}-name`}
                  className={`form-control ${getFieldError(`source-${idx}-name`) ? 'error' : ''}`}
                  value={src.name}
                  placeholder="Source name"
                  onChange={e => updateSource(idx, 'name', e.target.value)}
                  aria-invalid={!!getFieldError(`source-${idx}-name`)}
                  aria-describedby={getFieldError(`source-${idx}-name`) ? `error-${idx}-name` : undefined}
                />
                {getFieldError(`source-${idx}-name`) && (
                  <span id={`error-${idx}-name`} className="field-error" role="alert">
                    {getFieldError(`source-${idx}-name`)}
                  </span>
                )}
              </div>
              
              <div className="source-field source-url">
                <label htmlFor={`source-${idx}-url`} className="sr-only">Source URL</label>
                <input
                  id={`source-${idx}-url`}
                  className={`form-control ${getFieldError(`source-${idx}-url`) ? 'error' : ''}`}
                  value={src.url}
                  placeholder="RSS Feed URL"
                  onChange={e => updateSource(idx, 'url', e.target.value)}
                  type="url"
                  aria-invalid={!!getFieldError(`source-${idx}-url`)}
                  aria-describedby={getFieldError(`source-${idx}-url`) ? `error-${idx}-url` : undefined}
                />
                {getFieldError(`source-${idx}-url`) && (
                  <span id={`error-${idx}-url`} className="field-error" role="alert">
                    {getFieldError(`source-${idx}-url`)}
                  </span>
                )}
              </div>
              
              <button 
                type="button" 
                className="button delete-source"
                onClick={() => removeSource(idx)}
                aria-label={`Remove source ${src.name || 'unnamed'}`}
              >
                Remove
              </button>
            </div>
          ))}
        </section>

        <section className="card">
          <h3>AI Settings</h3>
          
          <div className="form-group">
            <label htmlFor="preferred-language">Preferred Language</label>
            <input
              id="preferred-language"
              className="form-control"
              value={config.global.preferred_language}
              onChange={e => setConfig({ 
                ...config, 
                global: { ...config.global, preferred_language: e.target.value } 
              })}
              placeholder="e.g., en, zh, ja"
            />
          </div>

          <div className="form-group">
            <label htmlFor="ai-provider">AI Provider</label>
            <select
              id="ai-provider"
              className="form-control"
              value={config.global.ai?.provider || ''}
              onChange={e => setConfig({ 
                ...config, 
                global: { 
                  ...config.global, 
                  ai: { 
                    ...config.global.ai, 
                    provider: e.target.value as 'gemini' | 'qwen' 
                  } 
                } 
              })}
            >
              <option value="">Auto (first available)</option>
              <option value="gemini">Gemini</option>
              <option value="qwen">Qwen</option>
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="ai-model">Model (optional)</label>
            <input
              id="ai-model"
              className="form-control"
              value={config.global.ai?.model || ''}
              onChange={e => setConfig({ 
                ...config, 
                global: { 
                  ...config.global, 
                  ai: { 
                    ...config.global.ai!, 
                    model: e.target.value 
                  } 
                } 
              })}
              placeholder="e.g., gemini-2.0-flash or qwen-turbo"
              aria-describedby="ai-model-help"
            />
            <span id="ai-model-help" className="help-text">
              Leave empty to use the default model for the selected provider
            </span>
          </div>
        </section>
      </form>
    </div>
  )
}

export default ConfigForm
