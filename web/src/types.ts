// Shared types for Sieve frontend

export interface Item {
  ID: string
  Source: string
  Title: string
  Link: string
  Description: string
  Summary: string
  InterestLevel: 'high_interest' | 'interest' | 'uninterested' | 'exclude'
  IsRead: boolean
  Reason: string
  Thought: string
  Saved?: boolean
  SavedAt?: string
  UserInterestOverride?: 'high_interest' | 'interest' | 'uninterested' | 'exclude'
  DuplicateOf?: string
  PublishedAt?: string
}

export interface Source {
  name: string
  title: string
  url: string
  high_interest?: string
  interest?: string
  uninterested?: string
  exclude?: string
  plugins?: string[]
  summarize?: boolean
  timeout?: number
  ai?: AIConfig
}

export interface AIConfig {
  provider: 'gemini' | 'qwen'
  model?: string
}

export interface GlobalConfig {
  high_interest: string
  interest: string
  uninterested: string
  exclude: string
  preferred_language: string
  timeout?: number
  ai?: AIConfig
  ai_time_between_ms?: number
  ai_burst_limit?: number
  ai_max_concurrency?: number
}

export interface Config {
  $schema?: string
  global: GlobalConfig
  sources: Source[]
}

export interface ApiError {
  message: string
  status: number
}

export type LoadingState = 'idle' | 'loading' | 'success' | 'error'

export interface AsyncState<T> {
  data: T | null
  state: LoadingState
  error: string | null
}

export interface ItemStats {
  total_visible: number
  saved: number
  high_interest: number
  unread_visible: number
  interest: number
  uninterested: number
}

export interface SourceStats {
  source: string
  visible: number
  saved: number
  high_interest: number
}

export interface SourceSuggestion {
  source: string
  visible: number
  reason: string
}

export interface Feed {
  id: string
  name: string
  url: string
  enabled: boolean
  high_interest?: string
  interest?: string
  uninterested?: string
  exclude?: string
  plugins?: string[]
  summarize?: boolean
  timeout?: number
  ai_provider?: string
  ai_model?: string
}

export interface Settings {
  high_interest?: string
  interest?: string
  uninterested?: string
  exclude?: string
  preferred_language?: string
  ai_provider?: string
  ai_model?: string
  ai_time_between_ms?: string
  ai_burst_limit?: string
  ai_max_concurrency?: string
}
