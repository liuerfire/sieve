import React, { useCallback, useMemo, useState } from 'react'
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

const formatPublishedDate = (value?: string) => {
  if (!value) return null

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null

  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }).format(date)
}

const ItemCard: React.FC<ItemCardProps> = ({ item, onUpdate }) => {
  const [showActions, setShowActions] = useState(false)
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
  const publishedDate = useMemo(() => formatPublishedDate(item.PublishedAt), [item.PublishedAt])

  const updateLevel = useCallback(async (level: string) => {
    if (isUpdating) return

    setIsUpdating(true)
    setError(null)

    const previousLevel = optimisticLevel || item.InterestLevel
    setOptimisticLevel(level)
    setOptimisticOverride(level)

    try {
      await api.updateItem(item.ID, { level, user_interest_override: level })
      onUpdate()
    } catch (err) {
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
    setOptimisticRead(newReadState)

    try {
      await api.updateItem(item.ID, { read: newReadState })
      onUpdate()
    } catch (err) {
      setOptimisticRead(null)
      setError(err instanceof Error ? err.message : 'Failed to update read status')
    } finally {
      setIsUpdating(false)
    }
  }, [item.ID, displayRead, isUpdating, onUpdate])

  const deleteItem = useCallback(async () => {
    if (isUpdating) return

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

  const sanitizedSummary = item.Summary ? DOMPurify.sanitize(item.Summary) : ''

  return (
    <article className={`card story-card ${displayRead ? 'read' : ''} ${isUpdating ? 'updating' : ''}`}>
      {error && (
        <div className="error-banner" role="alert">
          {error}
          <button onClick={() => setError(null)} aria-label="Dismiss error">×</button>
        </div>
      )}

      <div className={`story-row ${showActions ? 'expanded' : 'collapsed'}`}>
        <div className="story-main">
          <div className="story-main-top">
            <span
              className={`level-badge level-${displayLevel}`}
              aria-label={`Interest level: ${levelLabels[displayLevel] || displayLevel}`}
            >
              {levelLabels[displayLevel] || displayLevel.replace('_', ' ')}
            </span>
          </div>

          <div className="story-meta-rail">
            <span>{item.Source}</span>
            {publishedDate && <span>{publishedDate}</span>}
            {displaySaved && <span>Saved</span>}
          </div>

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

          <div className="story-analysis">
            {item.Reason && (
              <p>
                <strong>AI Reason:</strong> {item.Reason}
              </p>
            )}
            {item.Thought && (
              <p>
                <strong>AI Thought:</strong> {item.Thought}
              </p>
            )}
          </div>

          {sanitizedSummary && (
            <div
              className="card-summary"
              dangerouslySetInnerHTML={{ __html: sanitizedSummary }}
            />
          )}
        </div>

        <button
          className="story-actions-handle"
          type="button"
          onClick={() => setShowActions((value) => !value)}
          aria-expanded={showActions}
          aria-controls={`story-actions-panel-${item.ID}`}
          aria-label={showActions ? 'Collapse controls' : 'Expand controls'}
        >
          {showActions ? '>>' : '<<'}
        </button>

        {showActions && (
          <div className="story-actions">
            <div id={`story-actions-panel-${item.ID}`} className="story-actions-panel">
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
                  className="button button-outline"
                  onClick={toggleSaved}
                  disabled={isUpdating}
                  aria-label={displaySaved ? 'Unsave item' : 'Save item'}
                >
                  {displaySaved ? 'Unsave' : 'Save'}
                </button>
                <button
                  className="button button-outline"
                  onClick={toggleRead}
                  disabled={isUpdating}
                  aria-label={displayRead ? 'Mark as unread' : 'Mark as read'}
                >
                  {displayRead ? 'Mark unread' : 'Mark read'}
                </button>
                <button
                  className="button button-outline delete-button"
                  onClick={deleteItem}
                  disabled={isUpdating}
                  aria-label="Delete item"
                >
                  Delete
                </button>
              </div>

              <label className="override-control" htmlFor={`override-${item.ID}`}>
                <span>Override</span>
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
              </label>
            </div>
          </div>
        )}
      </div>
    </article>
  )
}

export default ItemCard
