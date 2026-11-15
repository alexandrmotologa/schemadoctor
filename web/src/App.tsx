import React, { useState, useEffect } from 'react'
import { Header } from './components/Header'
import { StatsGrid } from './components/StatsGrid'
import { LiveStream } from './components/LiveStream'
import { SchemaPlayground } from './components/SchemaPlayground'
import { SchemaManager } from './components/SchemaManager'
import { MetricsSnapshot, RepairEvent, SchemaMeta } from './types'

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'dashboard' | 'playground' | 'schemas'>('dashboard')
  const [isConnected, setIsConnected] = useState<boolean>(false)

  const [stats, setStats] = useState<MetricsSnapshot>({
    total_requests: 0,
    repaired_requests: 0,
    valid_pass_through: 0,
    failed_repairs: 0,
    success_rate: 100,
    tokens_saved_estimated: 0,
    avg_latency_us: 0,
    truncation_fixed: 0,
    markdown_stripped: 0,
    quotes_fixed: 0,
    trailing_commas_fixed: 0,
    dangling_keys_fixed: 0,
    type_coercions: 0,
  })

  const [events, setEvents] = useState<RepairEvent[]>([])
  const [schemas, setSchemas] = useState<SchemaMeta[]>([])

  const fetchStats = async () => {
    try {
      const resp = await fetch('/api/stats')
      if (resp.ok) {
        const data = await resp.json()
        setStats(data)
      }
    } catch (e) {
      // Ignore network errors during polling
    }
  }

  const fetchLogs = async () => {
    try {
      const resp = await fetch('/api/logs?limit=50')
      if (resp.ok) {
        const data = await resp.json()
        if (data.events) {
          setEvents(data.events)
        }
      }
    } catch (e) {
      // Ignore
    }
  }

  const fetchSchemas = async () => {
    try {
      const resp = await fetch('/api/schemas')
      if (resp.ok) {
        const data = await resp.json()
        setSchemas(data)
      }
    } catch (e) {
      // Ignore
    }
  }

  useEffect(() => {
    fetchStats()
    fetchLogs()
    fetchSchemas()

    // Setup live SSE stream
    let eventSource: EventSource | null = null

    const connectSSE = () => {
      eventSource = new EventSource('/api/stream')

      eventSource.onopen = () => {
        setIsConnected(true)
      }

      eventSource.onmessage = (e) => {
        try {
          const newEvent: RepairEvent = JSON.parse(e.data)
          if (newEvent && newEvent.trace_id) {
            setEvents((prev) => [newEvent, ...prev.slice(0, 99)])
            fetchStats()
          }
        } catch (err) {
          // Keep stream open
        }
      }

      eventSource.onerror = () => {
        setIsConnected(false)
        eventSource?.close()
        setTimeout(connectSSE, 3000)
      }
    }

    connectSSE()

    // Polling fallback every 5 seconds
    const interval = setInterval(() => {
      fetchStats()
    }, 5000)

    return () => {
      clearInterval(interval)
      eventSource?.close()
    }
  }, [])

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans">
      <Header
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        isConnected={isConnected}
      />

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {activeTab === 'dashboard' && (
          <div className="space-y-8">
            <StatsGrid stats={stats} />
            <LiveStream events={events} onRefresh={() => { fetchStats(); fetchLogs(); }} />
          </div>
        )}

        {activeTab === 'playground' && (
          <SchemaPlayground schemas={schemas} />
        )}

        {activeTab === 'schemas' && (
          <SchemaManager schemas={schemas} onSchemaAdded={fetchSchemas} />
        )}
      </main>

      <footer className="border-t border-slate-900 bg-slate-950/80 py-4 text-center text-xs text-slate-500 font-mono">
        schemadoctor &bull; Zero-Latency JSON Self-Healing & AST Repair Proxy &bull; Sub-millisecond wire latency
      </footer>
    </div>
  )
}
