import React, { useState } from 'react'
import { RepairEvent } from '../types'
import { DiffViewer } from './DiffViewer'
import { CheckCircle2, Wrench, AlertTriangle, Eye, RefreshCw } from 'lucide-react'

interface LiveStreamProps {
  events: RepairEvent[]
  onRefresh?: () => void
}

export const LiveStream: React.FC<LiveStreamProps> = ({ events, onRefresh }) => {
  const [selectedEvent, setSelectedEvent] = useState<RepairEvent | null>(null)
  const [filter, setFilter] = useState<'all' | 'healed' | 'valid' | 'failed'>('all')

  const filtered = events.filter(e => {
    if (filter === 'all') return true
    return e.status === filter
  })

  return (
    <div className="space-y-4">
      {/* Table header and controls */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div className="flex items-center space-x-2">
          <span className="text-sm font-semibold text-slate-200">Intercepted LLM Streams</span>
          <span className="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-mono">
            {events.length} events
          </span>
        </div>

        <div className="flex items-center space-x-2">
          {/* Status filters */}
          <div className="flex bg-slate-950 p-1 rounded-lg border border-slate-800 text-xs">
            {(['all', 'healed', 'valid', 'failed'] as const).map(tab => (
              <button
                key={tab}
                onClick={() => setFilter(tab)}
                className={`px-2.5 py-1 rounded capitalize font-medium transition-colors ${
                  filter === tab
                    ? 'bg-slate-800 text-white'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                {tab}
              </button>
            ))}
          </div>

          {onRefresh && (
            <button
              onClick={onRefresh}
              className="p-1.5 rounded-lg bg-slate-900 hover:bg-slate-800 border border-slate-800 text-slate-400 hover:text-white transition-colors"
              title="Refresh"
            >
              <RefreshCw className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Selected event diff viewer */}
      {selectedEvent && (
        <div className="relative">
          <button
            onClick={() => setSelectedEvent(null)}
            className="absolute top-3 right-3 text-xs text-slate-400 hover:text-white z-10 px-2 py-0.5 rounded bg-slate-800 border border-slate-700"
          >
            Close Diff
          </button>
          <DiffViewer
            original={selectedEvent.raw_input}
            repaired={selectedEvent.repaired_output}
            flags={selectedEvent.flags}
            latencyUs={selectedEvent.latency_us}
            tokensSaved={selectedEvent.tokens_saved}
          />
        </div>
      )}

      {/* Events table */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-slate-950/80 text-slate-400 font-mono uppercase tracking-wider border-b border-slate-800">
              <tr>
                <th className="py-3 px-4">Status</th>
                <th className="py-3 px-4">Trace ID</th>
                <th className="py-3 px-4">Model</th>
                <th className="py-3 px-4">Latency</th>
                <th className="py-3 px-4">Repairs Applied</th>
                <th className="py-3 px-4">Tokens Saved</th>
                <th className="py-3 px-4 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 font-sans">
              {filtered.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-8 text-center text-slate-500">
                    No intercepted requests yet. Send requests to the proxy to view live telemetry.
                  </td>
                </tr>
              ) : (
                filtered.map(evt => (
                  <tr
                    key={evt.id || evt.trace_id}
                    onClick={() => setSelectedEvent(evt)}
                    className="hover:bg-slate-800/40 cursor-pointer transition-colors"
                  >
                    <td className="py-3 px-4 whitespace-nowrap">
                      {evt.status === 'healed' ? (
                        <span className="inline-flex items-center space-x-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20">
                          <Wrench className="w-3 h-3" />
                          <span>Healed</span>
                        </span>
                      ) : evt.status === 'valid' ? (
                        <span className="inline-flex items-center space-x-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                          <CheckCircle2 className="w-3 h-3" />
                          <span>Valid</span>
                        </span>
                      ) : (
                        <span className="inline-flex items-center space-x-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-rose-500/10 text-rose-400 border border-rose-500/20">
                          <AlertTriangle className="w-3 h-3" />
                          <span>Failed</span>
                        </span>
                      )}
                    </td>

                    <td className="py-3 px-4 font-mono text-slate-300">
                      {evt.trace_id}
                    </td>

                    <td className="py-3 px-4 text-slate-300">
                      <span className="px-1.5 py-0.5 rounded bg-slate-800 font-mono text-[11px] text-slate-300 border border-slate-700">
                        {evt.model}
                      </span>
                    </td>

                    <td className="py-3 px-4 font-mono text-emerald-400">
                      {(evt.latency_us / 1000).toFixed(3)} ms
                    </td>

                    <td className="py-3 px-4 text-slate-400 max-w-xs truncate">
                      {evt.flags || 'None (Valid JSON)'}
                    </td>

                    <td className="py-3 px-4 font-mono text-amber-400">
                      {evt.tokens_saved > 0 ? `+${evt.tokens_saved}` : '-'}
                    </td>

                    <td className="py-3 px-4 text-right">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          setSelectedEvent(evt)
                        }}
                        className="inline-flex items-center space-x-1 text-slate-400 hover:text-blue-400 transition-colors"
                      >
                        <Eye className="w-3.5 h-3.5" />
                        <span>Inspect</span>
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
