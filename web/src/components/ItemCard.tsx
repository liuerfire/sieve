import React, { useState, useCallback } from 'react'
import DOMPurify from 'dompurify'
import type { Item } from '../types'
import { api } from '../api'

interface ItemCardProps {
  item: Item
  onUpdate: () => void
}

const levelLabels: Record<string, string> = {
  high_interest: 'High Interest',
  interest: 'Interest',
  uninterested: 'Uninterested',
  exclude: 'Exclude',
}

const levelEmoji: Record<string, string> = {
  high_interest: '⭐⭐',
  interest: '⭐',
  uninterested: '⚪',
  exclude: '🚫',
}

const ItemCard: React.FC<ItemCardProps> = ({ item, onUpdate }) => {
  const [showThought, setShowThought] = useState(true)
  const [isUpdating, setIsUpdating] = useState(false)
  const [optimisticRead, setOptimisticRead] = useState<boolean | null>(null)
  const [optimisticLevel, setOptimisticLevel] = useState<string | null>(null)
  const [optimisticSaved, setOptimisticSaved] = useState<boolean | null>(null)
  const [optimisticOverride, setOptimisticOverride] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const displayRead = optimisticRead !== null ? optimisticRead : item.IsRead
  const displaySaved = optimisticSaved !== null ? optimisticSaved : Boolean(item.Saved)
  const baseLevel = optimisticLevel !== null ? optimisticLevel : item.InterestLevel
  const overrideLevel = optimisticOverride !== null ? optimisticOverride : item.UserInterestOverride
  const displayLevel = overrideLevel || baseLevel

  const updateLevel = useCallback(async (level: string) => {
    if (isUpdating) return
    
    setIsUpdating(true)
    setError(null)
    
    // Optimistic update
    const previousLevel = optimisticLevel || item.InterestLevel
    setOptimisticLevel(level)
    setOptimisticOverride(level)
    
    try {
      await api.updateItem(item.ID, { level, user_interest_override: level })
      onUpdate()
    } catch (err) {
      // Rollback on error
      setOptimisticLevel(previousLevel)
      setOptimisticOverride(item.UserInterestOverride || '')
      setError(err instanceof Error ? err.message : 'Failed to update level')
    } finally {
      setIsUpdating(false)
    }
  }, [item.ID, item.InterestLevel, item.UserInterestOverride, optimisticLevel, isUpdating, onUpdate])

  const toggleSaved = useCallback(async () => {
    if (isUpdating) return

    setIsUpdating(true)
    setError(null)
    const next = !displaySaved
    setOptimisticSaved(next)
    try {
      await api.updateItem(item.ID, { saved: next })
      onUpdate()
    } catch (err) {
      setOptimisticSaved(null)
      setError(err instanceof Error ? err.message : 'Failed to update saved status')
    } finally {
      setIsUpdating(false)
    }
  }, [displaySaved, isUpdating, item.ID, onUpdate])

  const setOverride = useCallback(async (value: string) => {
    if (isUpdating) return

    setIsUpdating(true)
    setError(null)
    const next = value || ''
    const previous = optimisticOverride !== null ? optimisticOverride : item.UserInterestOverride || ''
    setOptimisticOverride(next)
    try {
      await api.updateItem(item.ID, { user_interest_override: next })
      onUpdate()
    } catch (err) {
      setOptimisticOverride(previous)
      setError(err instanceof Error ? err.message : 'Failed to update interest override')
    } finally {
      setIsUpdating(false)
    }
  }, [isUpdating, item.ID, item.UserInterestOverride, onUpdate, optimisticOverride])

  const toggleRead = useCallback(async () => {
    if (isUpdating) return
    
    setIsUpdating(true)
    setError(null)
    
    const newReadState = !displayRead
    
    // Optimistic update
    setOptimisticRead(newReadState)
    
    try {
      await api.updateItem(item.ID, { read: newReadState })
      onUpdate()
    } catch (err) {
      // Rollback on error
      setOptimisticRead(null)
      setError(err instanceof Error ? err.message : 'Failed to update read status')
    } finally {
      setIsUpdating(false)
    }
  }, [item.ID, displayRead, isUpdating, onUpdate])

  const deleteItem = useCallback(async () => {
    if (isUpdating) return
    
    // Use custom modal instead of confirm() for better UX
    if (!window.confirm('Are you sure you want to delete this item?')) {
      return
    }
    
    setIsUpdating(true)
    setError(null)
    
    try {
      await api.deleteItem(item.ID)
      onUpdate()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete item')
    } finally {
      setIsUpdating(false)
    }
  }, [item.ID, isUpdating, onUpdate])

  // Sanitize HTML content
  const sanitizedSummary = item.Summary 
    ? DOMPurify.sanitize(item.Summary)
    : item.Description 
      ? DOMPurify.sanitize(item.Description)
      : ''

  return (
    <div className={`card ${displayRead ? 'read' : ''} ${isUpdating ? 'updating' : ''}`}>
      {error && (
        <div className="error-banner" role="alert">
          {error}
          <button onClick={() => setError(null)} aria-label="Dismiss error">×</button>
        </div>
      )}
      
      <div className="card-header">
        <h3 className="card-title">
          <a 
            href={item.Link} 
            target="_blank" 
            rel="noopener noreferrer"
            aria-label={`${item.Title} (opens in new tab)`}
          >
            {item.Title}
          </a>
        </h3>
        <span 
          className={`level-badge level-${displayLevel}`}
          aria-label={`Interest level: ${levelLabels[displayLevel] || displayLevel}`}
        >
          {levelLabels[displayLevel] || displayLevel.replace('_', ' ')}
        </span>
      </div>

      <div className="card-meta">
        <span>{item.Source}</span>
        <button 
          className="button-outline" 
          onClick={() => setShowThought(!showThought)}
          aria-expanded={showThought}
          aria-controls={`thought-panel-${item.ID}`}
        >
          {showThought ? 'Hide Thought' : 'View Thought'}
        </button>
      </div>

      {showThought && (
        <div 
          id={`thought-panel-${item.ID}`}
          className="thought-panel"
        >
          <p><strong>Reason:</strong> {item.Reason}</p>
          <p><strong>AI Thought:</strong> {item.Thought}</p>
        </div>
      )}

      <div 
        className="card-summary"
        dangerouslySetInnerHTML={{ __html: sanitizedSummary }}
      />

      <div className="card-actions">
        <div className="level-controls" role="group" aria-label="Change interest level">
          {(Object.keys(levelEmoji) as Array<keyof typeof levelEmoji>).map((level) => (
            <button
              key={level}
              onClick={() => updateLevel(level)}
              disabled={isUpdating}
              title={levelLabels[level]}
              aria-label={`Mark as ${levelLabels[level]}`}
              aria-pressed={displayLevel === level}
              className={displayLevel === level ? 'active' : ''}
            >
              {levelEmoji[level]}
            </button>
          ))}
        </div>

        <div className="action-buttons">
          <button
            className="button-outline"
            onClick={toggleSaved}
            disabled={isUpdating}
            aria-label={displaySaved ? 'Unsave item' : 'Save item'}
          >
            {displaySaved ? 'Unsave' : 'Save'}
          </button>
          <button 
            className="button-outline" 
            onClick={toggleRead}
            disabled={isUpdating}
            aria-label={displayRead ? 'Mark as unread' : 'Mark as read'}
          >
            {displayRead ? 'Mark Unread' : 'Mark Read'}
          </button>
          <button 
            className="button-outline delete-button" 
            onClick={deleteItem}
            disabled={isUpdating}
            aria-label="Delete item"
          >
            Delete
          </button>
        </div>
      </div>

      <div className="card-meta">
        <label htmlFor={`override-${item.ID}`}>Override:</label>
        <select
          id={`override-${item.ID}`}
          className="form-control"
          value={overrideLevel || ''}
          onChange={e => setOverride(e.target.value)}
          disabled={isUpdating}
        >
          <option value="">Use AI result</option>
          <option value="high_interest">High Interest</option>
          <option value="interest">Interest</option>
          <option value="uninterested">Uninterested</option>
          <option value="exclude">Exclude</option>
        </select>
      </div>
    </div>
  )
}

export default ItemCard
