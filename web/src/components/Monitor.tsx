import React, { useEffect, useState, useCallback } from 'react'
import { api, createEventSource } from '../api'
import type { ProgressEvent, AsyncState } from '../types'

interface RunState {
  isRunning: boolean
  startTime: Date | null
}

const Monitor: React.FC = () => {
  const [events, setEvents] = useState<ProgressEvent[]>([])
  const [runState, setRunState] = useState<RunState>({
    isRunning: false,
    startTime: null,
  })
  const [startRunState, setStartRunState] = useState<AsyncState<void>>({
    data: null,
    state: 'idle',
    error: null,
  })
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'disconnected' | 'reconnecting'>('disconnected')

  const startRun = useCallback(async () => {
    setStartRunState({ data: null, state: 'loading', error: null })
    
    try {
      await api.startRun()
      setStartRunState({ data: null, state: 'success', error: null })
      setRunState({ isRunning: true, startTime: new Date() })
      setEvents([])
    } catch (err) {
      setStartRunState({
        data: null,
        state: 'error',
        error: err instanceof Error ? err.message : 'Failed to start run',
      })
    }
  }, [])

  useEffect(() => {
    const cleanup = createEventSource({
      onMessage: (ev: ProgressEvent) => {
        setEvents(prev => [ev, ...prev].slice(0, 50))
        
        if (ev.Type === 'gen_done') {
          setRunState({ isRunning: false, startTime: null })
        } else if (ev.Type === 'source_start' && !runState.isRunning) {
          setRunState({ isRunning: true, startTime: new Date() })
        }
      },
      onError: () => {
        setConnectionStatus('disconnected')
      },
      onOpen: () => {
        setConnectionStatus('connected')
      },
      maxRetries: 5,
      retryDelay: 3000,
    })

    return cleanup
  }, [runState.isRunning])

  const getEventIcon = (type: string): string => {
    switch (type) {
      case 'source_start': return '📡'
      case 'source_done': return '✅'
      case 'item_start': return '📄'
      case 'item_done': return '✓'
      case 'gen_start': return '📝'
      case 'gen_done': return '🎉'
      default: return '•'
    }
  }

  const formatDuration = (start: Date | null): string => {
    if (!start) return ''
    const diff = Math.floor((Date.now() - start.getTime()) / 1000)
    if (diff < 60) return `${diff}s`
    return `${Math.floor(diff / 60)}m ${diff % 60}s`
  }

  return (
    <div className="monitor">
      <div className="monitor-header">
        <div className="run-controls">
          <button 
            onClick={startRun} 
            className="run-button"
            disabled={runState.isRunning || startRunState.state === 'loading'}
            aria-label="Start aggregator run"
          >
            {startRunState.state === 'loading' ? 'Starting...' : 'Run Aggregator'}
          </button>
          
          {runState.isRunning && (
            <span className="run-timer" aria-live="polite">
              Running for {formatDuration(runState.startTime)}
            </span>
          )}
        </div>

        <div className="connection-status" aria-label={`Connection status: ${connectionStatus}`}>
          <span className={`status-dot ${connectionStatus}`} aria-hidden="true" />
          {connectionStatus === 'connected' ? 'Live' : connectionStatus === 'reconnecting' ? 'Reconnecting...' : 'Disconnected'}
        </div>
      </div>

      {startRunState.state === 'error' && (
        <div className="error-message" role="alert">
          Failed to start: {startRunState.error}
        </div>
      )}

      <div className="logs-container" role="log" aria-label="Aggregation logs" aria-live="polite">
        {events.length === 0 ? (
          <div className="placeholder">
            {runState.isRunning ? 'Waiting for events...' : 'Ready to run. Click "Run Aggregator" to start.'}
          </div>
        ) : (
          events.map((ev, i) => (
            <div key={i} className={`log-line log-${ev.Type}`}>
              <span className="ev-icon" aria-hidden="true">{getEventIcon(ev.Type)}</span>
              <span className="ev-type">[{ev.Type}]</span>
              <span className="ev-source">{ev.Source}</span>
              {(ev.Message || ev.Item) && (
                <span className="ev-message">- {ev.Message || ev.Item}</span>
              )}
              {ev.Count > 0 && ev.Total > 0 && (
                <span className="ev-progress">({ev.Count}/{ev.Total})</span>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  )
}

export default Monitor
