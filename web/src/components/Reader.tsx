import React, { useEffect, useState, useCallback } from 'react'
import ItemCard from './ItemCard'
import { api } from '../api'
import type { Item, AsyncState } from '../types'

const Reader: React.FC = () => {
  const [itemsState, setItemsState] = useState<AsyncState<Item[]>>({
    data: null,
    state: 'idle',
    error: null,
  })

  const fetchItems = useCallback(async () => {
    setItemsState(prev => ({ ...prev, state: 'loading', error: null }))
    
    try {
      const data = await api.getItems()
      setItemsState({ data, state: 'success', error: null })
    } catch (err) {
      setItemsState({
        data: null,
        state: 'error',
        error: err instanceof Error ? err.message : 'Failed to fetch items',
      })
    }
  }, [])

  useEffect(() => {
    fetchItems()
  }, [fetchItems])

  const handleRefresh = useCallback(() => {
    fetchItems()
  }, [fetchItems])

  const handleUpdate = useCallback(() => {
    // Refresh items after an update
    fetchItems()
  }, [fetchItems])

  return (
    <div className="reader">
      <div className="reader-header">
        <h2>Recent News</h2>
        <button 
          className="button button-outline" 
          onClick={handleRefresh}
          disabled={itemsState.state === 'loading'}
          aria-label="Refresh news items"
        >
          {itemsState.state === 'loading' ? 'Loading...' : 'Refresh'}
        </button>
      </div>

      {itemsState.state === 'error' && (
        <div className="error-message" role="alert">
          <p>Error loading items: {itemsState.error}</p>
          <button onClick={handleRefresh} className="button button-primary">
            Try Again
          </button>
        </div>
      )}

      {itemsState.state === 'loading' && !itemsState.data && (
        <div className="loading-state">
          <div className="spinner" aria-label="Loading items" />
          <p>Loading news items...</p>
        </div>
      )}

      {itemsState.state === 'success' && itemsState.data?.length === 0 && (
        <div className="empty-state">
          <p>No items found. Run the aggregator to fetch news!</p>
          <button onClick={handleRefresh} className="button button-primary">
            Check Again
          </button>
        </div>
      )}

      {itemsState.data && itemsState.data.length > 0 && (
        <div className="items-list" role="feed" aria-label="News items">
          {itemsState.data.map((item, index) => (
            <ItemCard 
              key={item.ID} 
              item={item} 
              onUpdate={handleUpdate}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default Reader
