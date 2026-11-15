export interface DiagnosticRecord {
  line: number
  column: number
  rule: string
  description: string
}

export interface RepairFlags {
  stripped_markdown: boolean
  closed_open_string: boolean
  balanced_delimiters: boolean
  removed_trailing_comma: boolean
  completed_dangling_key: boolean
  normalized_ident: boolean
  fixed_unescaped_quote: boolean
}

export interface MetricsSnapshot {
  total_requests: number
  repaired_requests: number
  valid_pass_through: number
  failed_repairs: number
  success_rate: number
  tokens_saved_estimated: number
  avg_latency_us: number
  truncation_fixed: number
  markdown_stripped: number
  quotes_fixed: number
  trailing_commas_fixed: number
  dangling_keys_fixed: number
  type_coercions: number
}

export interface RepairEvent {
  id: number
  trace_id: string
  timestamp: string
  provider: string
  model: string
  status: 'healed' | 'valid' | 'failed'
  latency_us: number
  tokens_saved: number
  original_size: number
  repaired_size: number
  flags: string
  raw_input: string
  repaired_output: string
}

export interface SchemaMeta {
  name: string
  description?: string
  raw: any
}
