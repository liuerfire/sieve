import React, { useEffect, useState } from 'react'

interface ProgressEvent {
  Type: string
  Source: string
  Item: string
  Message: string
  Count: number
  Total: number
}

function Monitor() {
  const [events, setEvents] = useState<ProgressEvent[]>([])

  const startRun = () => {
    fetch('/api/run', { method: 'POST' })
      .catch(err => console.error('start run failed:', err))
  }

  useEffect(() => {
    const sse = new EventSource('/api/events')
    sse.onmessage = (e) => {
      try {
        const ev = JSON.parse(e.data)
        setEvents(prev => [ev, ...prev].slice(0, 10))
      } catch (err) {
        console.error('parse event failed:', err)
      }
    }
    return () => sse.close()
  }, [])

  return (
    <div className="monitor">
      <button onClick={startRun} className="run-button">Run Aggregator</button>
      <div className="logs">
        {events.length === 0 ? <div className="placeholder">Ready to run...</div> : 
          events.map((ev, i) => (
            <div key={i} className="log-line">
              <span className="ev-type">[{ev.Type}]</span> {ev.Source} - {ev.Message || ev.Item}
            </div>
          ))
        }
      </div>
    </div>
  )
}

export default Monitor
