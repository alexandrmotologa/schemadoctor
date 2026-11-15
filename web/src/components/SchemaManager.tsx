import React, { useState } from 'react'
import { SchemaMeta } from '../types'
import { Database, Plus, Check, AlertCircle, Code2 } from 'lucide-react'

interface SchemaManagerProps {
  schemas: SchemaMeta[]
  onSchemaAdded: () => void
}

export const SchemaManager: React.FC<SchemaManagerProps> = ({ schemas, onSchemaAdded }) => {
  const [name, setName] = useState('')
  const [schemaJson, setSchemaJson] = useState(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "name": { "type": "string" },
    "age": { "type": "integer" },
    "score": { "type": "number" },
    "active": { "type": "boolean" },
    "tags": { "type": "array", "items": { "type": "string" } }
  },
  "required": ["name", "age"]
}`)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [selectedSchema, setSelectedSchema] = useState<SchemaMeta | null>(schemas[0] || null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSuccess(null)

    if (!name.trim()) {
      setError('Schema name is required')
      return
    }

    let parsed: any
    try {
      parsed = JSON.parse(schemaJson)
    } catch (err: any) {
      setError('Schema definition must be valid JSON: ' + err.message)
      return
    }

    setLoading(true)
    try {
      const resp = await fetch('/api/schemas', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          schema: parsed,
        }),
      })

      if (!resp.ok) {
        const text = await resp.text()
        throw new Error(text || 'Failed to register schema')
      }

      setSuccess(`Schema "${name}" successfully registered!`)
      setName('')
      onSchemaAdded()
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow">
        <h2 className="text-sm font-semibold text-white flex items-center space-x-2">
          <Database className="w-4 h-4 text-blue-400" />
          <span>JSON Schema & Type Coercion Registry</span>
        </h2>
        <p className="text-xs text-slate-400 mt-0.5">
          Define target schemas to enable automated type coercion (e.g. string numbers to integers) during proxy interception.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Schemas List */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col shadow">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-semibold uppercase text-slate-400 font-mono tracking-wider">
              Registered Schemas
            </span>
            <span className="text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono">
              {schemas.length}
            </span>
          </div>

          <div className="space-y-2 flex-1 overflow-y-auto max-h-[460px]">
            {schemas.length === 0 ? (
              <p className="text-xs text-slate-500 py-4 text-center">
                No schemas registered yet. Add one using the form.
              </p>
            ) : (
              schemas.map((s) => (
                <button
                  key={s.name}
                  onClick={() => setSelectedSchema(s)}
                  className={`w-full text-left p-3 rounded-lg border text-xs transition-colors ${
                    selectedSchema?.name === s.name
                      ? 'bg-blue-600/10 border-blue-500/40 text-blue-300'
                      : 'bg-slate-950 border-slate-800 text-slate-300 hover:border-slate-700'
                  }`}
                >
                  <div className="font-semibold font-mono">{s.name}</div>
                  {s.description && (
                    <div className="text-[11px] text-slate-500 mt-1 truncate">
                      {s.description}
                    </div>
                  )}
                </button>
              ))
            )}
          </div>
        </div>

        {/* Selected Schema Viewer / Add New Schema */}
        <div className="lg:col-span-2 space-y-6">
          {/* Add New Schema Form */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow">
            <h3 className="text-xs font-semibold uppercase text-slate-400 font-mono tracking-wider mb-4 flex items-center space-x-1.5">
              <Plus className="w-3.5 h-3.5 text-blue-400" />
              <span>Register New Schema</span>
            </h3>

            {error && (
              <div className="mb-4 bg-rose-950/40 border border-rose-800 rounded-lg p-3 text-xs text-rose-300 flex items-center space-x-2">
                <AlertCircle className="w-4 h-4 text-rose-400" />
                <span>{error}</span>
              </div>
            )}
            {success && (
              <div className="mb-4 bg-emerald-950/40 border border-emerald-800 rounded-lg p-3 text-xs text-emerald-300 flex items-center space-x-2">
                <Check className="w-4 h-4 text-emerald-400" />
                <span>{success}</span>
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">
                  Schema Identifier (Name)
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. user_profile, order_response, agent_plan"
                  className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-blue-500 font-mono"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">
                  JSON Schema Definition
                </label>
                <textarea
                  value={schemaJson}
                  onChange={(e) => setSchemaJson(e.target.value)}
                  rows={8}
                  className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 font-mono text-xs text-slate-200 focus:outline-none focus:border-blue-500 leading-relaxed"
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="flex items-center space-x-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-medium text-xs px-4 py-2 rounded-lg transition-colors shadow-sm"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>{loading ? 'Compiling...' : 'Register Schema'}</span>
              </button>
            </form>
          </div>

          {/* Active Schema JSON Preview */}
          {selectedSchema && (
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-semibold uppercase text-slate-400 font-mono tracking-wider flex items-center space-x-1.5">
                  <Code2 className="w-3.5 h-3.5 text-blue-400" />
                  <span>Schema: {selectedSchema.name}</span>
                </span>
              </div>
              <pre className="bg-slate-950 border border-slate-800 rounded-lg p-3 font-mono text-xs text-slate-300 overflow-x-auto max-h-[240px]">
                {JSON.stringify(selectedSchema.raw, null, 2)}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
