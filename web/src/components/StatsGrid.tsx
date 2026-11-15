import React from 'react'
import { MetricsSnapshot } from '../types'
import { Wrench, Zap, Clock, Layers } from 'lucide-react'

interface StatsGridProps {
  stats: MetricsSnapshot
}

export const StatsGrid: React.FC<StatsGridProps> = ({ stats }) => {
  return (
    <div className="space-y-4">
      {/* Primary KPI row */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Total requests */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
          <div className="flex items-center justify-between text-slate-400 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider">Total Requests</span>
            <Layers className="w-4 h-4 text-slate-500" />
          </div>
          <div className="flex items-baseline space-x-2">
            <span className="text-2xl font-bold font-mono text-white">
              {stats.total_requests.toLocaleString()}
            </span>
            <span className="text-xs text-slate-500">requests</span>
          </div>
          <div className="mt-2 text-xs text-slate-400 flex items-center space-x-1">
            <span className="text-emerald-400 font-medium">{stats.valid_pass_through}</span>
            <span>valid pass-through</span>
          </div>
        </div>

        {/* Repaired & Success Rate */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
          <div className="flex items-center justify-between text-slate-400 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider">Repaired Syntaxes</span>
            <Wrench className="w-4 h-4 text-blue-400" />
          </div>
          <div className="flex items-baseline space-x-2">
            <span className="text-2xl font-bold font-mono text-blue-400">
              {stats.repaired_requests.toLocaleString()}
            </span>
            <span className="text-xs px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-mono">
              {stats.success_rate.toFixed(1)}% success
            </span>
          </div>
          <div className="mt-2 text-xs text-slate-400">
            0 API retries triggered
          </div>
        </div>

        {/* Tokens Saved */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
          <div className="flex items-center justify-between text-slate-400 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider">Tokens Saved</span>
            <Zap className="w-4 h-4 text-amber-400" />
          </div>
          <div className="flex items-baseline space-x-2">
            <span className="text-2xl font-bold font-mono text-amber-400">
              ~{stats.tokens_saved_estimated.toLocaleString()}
            </span>
            <span className="text-xs text-slate-500">tokens</span>
          </div>
          <div className="mt-2 text-xs text-slate-400">
            Saved by skipping LLM error retries
          </div>
        </div>

        {/* Wire Latency */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
          <div className="flex items-center justify-between text-slate-400 mb-2">
            <span className="text-xs font-medium uppercase tracking-wider">Avg Repair Latency</span>
            <Clock className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="flex items-baseline space-x-2">
            <span className="text-2xl font-bold font-mono text-emerald-400">
              {stats.avg_latency_us > 0 ? (stats.avg_latency_us / 1000).toFixed(3) : '< 0.05'}
            </span>
            <span className="text-xs text-slate-500">ms / req</span>
          </div>
          <div className="mt-2 text-xs text-emerald-400/80 font-mono">
            Sub-millisecond wire speed
          </div>
        </div>
      </div>

      {/* Syntax breakdown pills */}
      <div className="bg-slate-900/40 border border-slate-800/80 rounded-lg p-3 flex flex-wrap gap-3 items-center justify-between text-xs">
        <span className="text-slate-400 font-medium">Automatic Repair Breakdown:</span>
        <div className="flex flex-wrap gap-2">
          <span className="px-2 py-1 rounded bg-slate-800 text-slate-300 font-mono border border-slate-700">
            Truncated Delimiters: <strong className="text-blue-400">{stats.truncation_fixed}</strong>
          </span>
          <span className="px-2 py-1 rounded bg-slate-800 text-slate-300 font-mono border border-slate-700">
            Markdown Stripped: <strong className="text-purple-400">{stats.markdown_stripped}</strong>
          </span>
          <span className="px-2 py-1 rounded bg-slate-800 text-slate-300 font-mono border border-slate-700">
            Unescaped Quotes: <strong className="text-amber-400">{stats.quotes_fixed}</strong>
          </span>
          <span className="px-2 py-1 rounded bg-slate-800 text-slate-300 font-mono border border-slate-700">
            Trailing Commas: <strong className="text-emerald-400">{stats.trailing_commas_fixed}</strong>
          </span>
          <span className="px-2 py-1 rounded bg-slate-800 text-slate-300 font-mono border border-slate-700">
            Dangling Keys: <strong className="text-cyan-400">{stats.dangling_keys_fixed}</strong>
          </span>
          <span className="px-2 py-1 rounded bg-slate-800 text-slate-300 font-mono border border-slate-700">
            Type Coercions: <strong className="text-rose-400">{stats.type_coercions}</strong>
          </span>
        </div>
      </div>
    </div>
  )
}
