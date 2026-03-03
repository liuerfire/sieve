import React, { useEffect, useState } from 'react'
import ItemCard from './ItemCard'

function Reader() {
  const [items, setItems] = useState<any[]>([])

  const fetchItems = () => {
    fetch('/api/items')
      .then(res => res.json())
      .then(data => {
        if (Array.isArray(data)) {
          setItems(data)
        }
      })
      .catch(err => console.error('fetch items failed:', err))
  }

  useEffect(() => {
    fetchItems()
  }, [])

  return (
    <div className="reader">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '32px' }}>
        <h2 style={{ margin: 0 }}>Recent News</h2>
        <button className="button button-outline" onClick={fetchItems}>Refresh</button>
      </div>
      
      {items.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '100px', color: '#64748b' }}>
          No items found. Run the aggregator to fetch news!
        </div>
      ) : (
        items.map(item => (
          <ItemCard key={item.ID} item={item} onUpdate={fetchItems} />
        ))
      )}
    </div>
  )
}

export default Reader
