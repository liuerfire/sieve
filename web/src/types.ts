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
  html_max_age_days?: number
  enable_archives?: boolean
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
