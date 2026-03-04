import React, { useState } from 'react'
import Reader from './components/Reader'
import ConfigForm from './components/ConfigForm'
import './App.css'

function App() {
  const [view, setView] = useState('reader')

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
      </aside>

      <main className="main-content">
        {view === 'reader' && <Reader />}
        {view === 'config' && <ConfigForm />}
      </main>
    </div>
  )
}

export default App
