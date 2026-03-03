import React, { useState } from 'react'

interface Item {
  ID: string
  Title: string
  Source: string
  Summary: string
  Link: string
  InterestLevel: string
  IsRead: boolean
  Reason: string
  Thought: string
}

interface ItemCardProps {
  item: Item
  onUpdate: () => void
}

const ItemCard: React.FC<ItemCardProps> = ({ item, onUpdate }) => {
  const [showThought, setShowThought] = useState(false)

  const updateLevel = (level: string) => {
    fetch(`/api/items/${item.ID}`, {
      method: 'PATCH',
      body: JSON.stringify({ level })
    }).then(onUpdate)
  }

  const toggleRead = () => {
    fetch(`/api/items/${item.ID}`, {
      method: 'PATCH',
      body: JSON.stringify({ read: !item.IsRead })
    }).then(onUpdate)
  }

  const deleteItem = () => {
    if (confirm('Are you sure you want to delete this item?')) {
      fetch(`/api/items/${item.ID}`, { method: 'DELETE' }).then(onUpdate)
    }
  }

  return (
    <div className={`card ${item.IsRead ? 'read' : ''}`}>
      <div className="card-header">
        <h3 className="card-title">
          <a href={item.Link} target="_blank" rel="noopener noreferrer">{item.Title}</a>
        </h3>
        <span className={`level-badge level-${item.InterestLevel}`}>
          {item.InterestLevel.replace('_', ' ')}
        </span>
      </div>
      
      <div className="card-meta">
        <span>{item.Source}</span>
        <button className="button-outline" onClick={() => setShowThought(!showThought)}>
          {showThought ? 'Hide Thought' : 'View Thought'}
        </button>
      </div>

      {showThought && (
        <div className="thought-panel" style={{ background: '#f8fafc', padding: '12px', borderRadius: '6px', marginBottom: '16px', fontSize: '0.875rem' }}>
          <strong>Reason:</strong> {item.Reason}
          <br />
          <strong>AI Thought:</strong> {item.Thought}
        </div>
      )}

      <div className="card-summary">{item.Summary}</div>

      <div className="card-actions">
        <div className="level-controls" style={{ display: 'flex', gap: '8px' }}>
          <button onClick={() => updateLevel('high_interest')} title="High Interest">⭐⭐</button>
          <button onClick={() => updateLevel('interest')} title="Interest">⭐</button>
          <button onClick={() => updateLevel('uninterested')} title="Uninterested">⚪</button>
          <button onClick={() => updateLevel('exclude')} title="Exclude">🚫</button>
        </div>
        
        <div style={{ display: 'flex', gap: '12px' }}>
          <button className="button-outline" onClick={toggleRead}>
            {item.IsRead ? 'Mark Unread' : 'Mark Read'}
          </button>
          <button className="button-outline" style={{ color: '#ef4444' }} onClick={deleteItem}>
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}

export default ItemCard
