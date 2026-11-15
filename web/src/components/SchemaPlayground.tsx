import React, { useState } from 'react'
import { SchemaMeta, DiagnosticRecord } from '../types'
import { DiffViewer } from './DiffViewer'
import { Play, Sparkles, AlertCircle, RefreshCw, FileCode, Check, Database } from 'lucide-react'

interface SchemaPlaygroundProps {
  schemas: SchemaMeta[]
  onSchemaAdded?: () => void
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

export const SchemaPlayground: React.FC<SchemaPlaygroundProps> = ({ schemas, onSchemaAdded }) => {
  const [input, setInput] = useState<string>(PRESETS[0].value)
  const [selectedSchema, setSelectedSchema] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)
  const [inferring, setInferring] = useState<boolean>(false)
  const [inferredSchema, setInferredSchema] = useState<any | null>(null)
  const [schemaNameInput, setSchemaNameInput] = useState<string>('auto_inferred_schema')
  const [saveSuccess, setSaveSuccess] = useState<string | null>(null)

  const [result, setResult] = useState<{
    repaired: string
    valid: boolean
    flags: any
    diagnostics?: DiagnosticRecord[]
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
    setInferredSchema(null)
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

  const handleInferSchema = async () => {
    setInferring(true)
    setError(null)
    setSaveSuccess(null)
    try {
      const resp = await fetch('/api/schemas/infer', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          input: input,
          title: 'InferredSchema',
        }),
      })

      if (!resp.ok) {
        throw new Error('Failed to infer JSON schema')
      }

      const data = await resp.json()
      setInferredSchema(data.schema)
    } catch (err: any) {
      setError(err.message || 'Failed to infer schema')
    } finally {
      setInferring(false)
    }
  }

  const handleSaveInferredSchema = async () => {
    if (!inferredSchema || !schemaNameInput.trim()) return
    try {
      const resp = await fetch('/api/schemas', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: schemaNameInput.trim(),
          schema: inferredSchema,
        }),
      })
      if (!resp.ok) throw new Error('Failed to register schema')
      setSaveSuccess(`Schema "${schemaNameInput.trim()}" successfully registered!`)
      if (onSchemaAdded) onSchemaAdded()
      setTimeout(() => setSaveSuccess(null), 3000)
    } catch (e: any) {
      setError(e.message)
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

            {/* Infer Schema Button */}
            <button
              onClick={handleInferSchema}
              disabled={inferring || !input.trim()}
              className="flex items-center space-x-1.5 bg-slate-800 hover:bg-slate-700 disabled:opacity-50 text-slate-200 font-medium text-xs px-3.5 py-2 rounded-lg transition-colors border border-slate-700 shadow-sm"
              title="Automatically generate a JSON Schema from this payload"
            >
              {inferring ? (
                <RefreshCw className="w-3.5 h-3.5 animate-spin text-blue-400" />
              ) : (
                <FileCode className="w-3.5 h-3.5 text-blue-400" />
              )}
              <span>Infer Schema</span>
            </button>

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

      {/* Inferred Schema Drawer */}
      {inferredSchema && (
        <div className="bg-slate-900 border border-blue-500/30 rounded-xl p-4 shadow-lg space-y-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div className="flex items-center space-x-2">
              <Database className="w-4 h-4 text-blue-400" />
              <span className="text-xs font-semibold text-white">Auto-Inferred JSON Schema</span>
            </div>

            <div className="flex items-center space-x-2">
              <input
                type="text"
                value={schemaNameInput}
                onChange={(e) => setSchemaNameInput(e.target.value)}
                placeholder="Schema identifier name"
                className="bg-slate-950 border border-slate-800 text-xs px-2.5 py-1.5 rounded-lg text-slate-200 font-mono focus:outline-none focus:border-blue-500"
              />
              <button
                onClick={handleSaveInferredSchema}
                className="flex items-center space-x-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs px-3 py-1.5 rounded-lg transition-colors font-medium shadow-sm"
              >
                <Check className="w-3.5 h-3.5" />
                <span>Save to Registry</span>
              </button>
              <button
                onClick={() => setInferredSchema(null)}
                className="text-xs text-slate-400 hover:text-white px-2 py-1"
              >
                Dismiss
              </button>
            </div>
          </div>

          {saveSuccess && (
            <div className="text-xs text-emerald-400 font-medium bg-emerald-500/10 border border-emerald-500/20 px-3 py-1.5 rounded-lg">
              {saveSuccess}
            </div>
          )}

          <pre className="bg-slate-950 border border-slate-800 p-3 rounded-lg text-xs font-mono text-slate-300 max-h-[180px] overflow-y-auto">
            {JSON.stringify(inferredSchema, null, 2)}
          </pre>
        </div>
      )}

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
              diagnostics={result.diagnostics}
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
