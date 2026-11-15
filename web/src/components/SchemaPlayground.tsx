import React, { useState } from 'react'
import { SchemaMeta } from '../types'
import { DiffViewer } from './DiffViewer'
import { Play, Sparkles, AlertCircle, RefreshCw } from 'lucide-react'

interface SchemaPlaygroundProps {
  schemas: SchemaMeta[]
}

const PRESETS: { label: string; value: string; schema?: string }[] = [
  {
    label: 'Truncated Array (max_tokens hit)',
    value: `{"status": "success", "results": [{"id": 1, "name": "Item A"}, {"id": 2, "name": "Item B"`,
  },
  {
    label: 'Markdown Code Fence + Chat Preamble',
    value: `Here is the JSON structure you requested for the configuration:
\`\`\`json
{
  "service": "auth-service",
  "active": True,
  "endpoints": ["/login", "/refresh", ],
}
\`\`\`
Let me know if you would like me to adjust the fields!`,
  },
  {
    label: 'Unescaped Nested Double Quotes',
    value: `{"user": "Alex", "review": "The customer said "This is amazing" and bought 2 items", "score": 5}`,
  },
  {
    label: 'Dangling Key + Trailing Commas',
    value: `{"id": 101, "title": "Setup", "metadata": , "tags": ["prod", "docker", ], }`,
  },
  {
    label: 'Python Literals (True / False / None)',
    value: `{"is_enabled": True, "debug_mode": False, "deleted_at": None, "priority": 1}`,
  },
  {
    label: 'Schema Type Coercion (String Numbers & Booleans)',
    value: `{"name": "Alice Developer", "age": "29", "score": "98.5", "active": "true", "tags": "golang"}`,
    schema: 'user_profile',
  },
]

export const SchemaPlayground: React.FC<SchemaPlaygroundProps> = ({ schemas }) => {
  const [input, setInput] = useState<string>(PRESETS[0].value)
  const [selectedSchema, setSelectedSchema] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)
  const [result, setResult] = useState<{
    repaired: string
    valid: boolean
    flags: any
    latency_us: number
    tokens_saved: number
  } | null>(null)
  const [error, setError] = useState<string | null>(null)

  const handleSelectPreset = (preset: typeof PRESETS[0]) => {
    setInput(preset.value)
    if (preset.schema) {
      setSelectedSchema(preset.schema)
    } else {
      setSelectedSchema('')
    }
    setResult(null)
    setError(null)
  }

  const handleRunRepair = async () => {
    setLoading(true)
    setError(null)
    try {
      const resp = await fetch('/api/repair', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          input: input,
          schema: selectedSchema,
        }),
      })

      if (!resp.ok) {
        throw new Error(`API returned HTTP ${resp.status}`)
      }

      const data = await resp.json()
      setResult(data)
    } catch (err: any) {
      setError(err.message || 'Repair request failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Controls header */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <h2 className="text-sm font-semibold text-white flex items-center space-x-2">
              <Sparkles className="w-4 h-4 text-blue-400" />
              <span>Interactive Repair Sandbox</span>
            </h2>
            <p className="text-xs text-slate-400 mt-0.5">
              Paste malformed LLM outputs or select failure presets to evaluate sub-millisecond AST healing.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {/* Presets dropdown */}
            <select
              onChange={(e) => {
                const idx = parseInt(e.target.value, 10)
                if (!isNaN(idx)) handleSelectPreset(PRESETS[idx])
              }}
              className="bg-slate-950 border border-slate-800 text-xs text-slate-300 rounded-lg px-3 py-2 font-medium focus:outline-none focus:border-blue-500"
            >
              <option value="">Load Failure Preset...</option>
              {PRESETS.map((p, i) => (
                <option key={i} value={i}>
                  {p.label}
                </option>
              ))}
            </select>

            {/* Target Schema Selector */}
            <select
              value={selectedSchema}
              onChange={(e) => setSelectedSchema(e.target.value)}
              className="bg-slate-950 border border-slate-800 text-xs text-slate-300 rounded-lg px-3 py-2 font-medium focus:outline-none focus:border-blue-500"
            >
              <option value="">Target Schema: None (Pure AST)</option>
              {schemas.map((s) => (
                <option key={s.name} value={s.name}>
                  Schema: {s.name}
                </option>
              ))}
            </select>

            {/* Run Button */}
            <button
              onClick={handleRunRepair}
              disabled={loading || !input.trim()}
              className="flex items-center space-x-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-medium text-xs px-4 py-2 rounded-lg transition-colors shadow-sm"
            >
              {loading ? (
                <RefreshCw className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Play className="w-3.5 h-3.5" />
              )}
              <span>Heal JSON</span>
            </button>
          </div>
        </div>
      </div>

      {/* Editor & Diff grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Input Textarea */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col shadow">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase text-slate-400 font-mono tracking-wider">
              Input (Malformed LLM Generation)
            </span>
            <span className="text-[11px] text-slate-500 font-mono">{input.length} chars</span>
          </div>
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            rows={12}
            placeholder="Paste malformed JSON or markdown code block here..."
            className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 font-mono text-xs text-slate-200 focus:outline-none focus:border-blue-500 resize-y flex-1 leading-relaxed selection:bg-blue-600"
          />
        </div>

        {/* Live Diff Preview */}
        <div className="flex flex-col">
          {error && (
            <div className="bg-rose-950/40 border border-rose-800 rounded-xl p-4 text-xs text-rose-300 flex items-center space-x-2 mb-4">
              <AlertCircle className="w-4 h-4 text-rose-400" />
              <span>{error}</span>
            </div>
          )}

          {result ? (
            <DiffViewer
              original={input}
              repaired={result.repaired}
              flags={typeof result.flags === 'object' ? Object.keys(result.flags).filter(k => result.flags[k]).join(', ') : undefined}
              latencyUs={result.latency_us}
              tokensSaved={result.tokens_saved}
            />
          ) : (
            <div className="bg-slate-900 border border-dashed border-slate-800 rounded-xl p-12 flex flex-col items-center justify-center text-center text-slate-500 flex-1 min-h-[300px]">
              <Sparkles className="w-8 h-8 text-slate-600 mb-3" />
              <p className="text-sm font-medium text-slate-400">Ready to Heal</p>
              <p className="text-xs text-slate-500 mt-1 max-w-sm">
                Click "Heal JSON" or choose a failure preset to see deterministic AST repairs and validation in action.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
