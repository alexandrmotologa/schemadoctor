import React, { useState } from 'react'
import { Check, Copy, AlertCircle, Sparkles, Terminal, ChevronDown, ChevronUp } from 'lucide-react'
import { DiagnosticRecord } from '../types'

interface DiffViewerProps {
  original: string
  repaired: string
  flags?: string
  latencyUs?: number
  tokensSaved?: number
  diagnostics?: DiagnosticRecord[]
}

export const DiffViewer: React.FC<DiffViewerProps> = ({
  original,
  repaired,
  flags,
  latencyUs,
  tokensSaved,
  diagnostics,
}) => {
  const [copied, setCopied] = useState(false)
  const [showDiagnostics, setShowDiagnostics] = useState(true)

  const handleCopy = () => {
    navigator.clipboard.writeText(repaired)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  // Format repaired JSON if valid
  let formattedRepaired = repaired
  try {
    const parsed = JSON.parse(repaired)
    formattedRepaired = JSON.stringify(parsed, null, 2)
  } catch (e) {
    // Keep as is
  }

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-lg">
      <div className="bg-slate-950 px-4 py-3 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Sparkles className="w-4 h-4 text-blue-400" />
          <span className="text-xs font-semibold text-slate-200 uppercase tracking-wider">
            AST Syntax Healer Diff
          </span>
          {latencyUs !== undefined && (
            <span className="text-xs px-2 py-0.5 rounded bg-blue-500/10 text-blue-400 font-mono border border-blue-500/20">
              {(latencyUs / 1000).toFixed(3)} ms
            </span>
          )}
          {tokensSaved !== undefined && tokensSaved > 0 && (
            <span className="text-xs px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 font-mono border border-amber-500/20">
              +{tokensSaved} tokens saved
            </span>
          )}
        </div>

        <button
          onClick={handleCopy}
          className="flex items-center space-x-1.5 px-2.5 py-1 text-xs font-medium text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 rounded border border-slate-700 transition-colors"
        >
          {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
          <span>{copied ? 'Copied' : 'Copy Healed JSON'}</span>
        </button>
      </div>

      {flags && (
        <div className="bg-slate-900/90 px-4 py-2 border-b border-slate-800 text-xs flex items-center space-x-2">
          <span className="text-slate-400">Repairs Applied:</span>
          <div className="flex flex-wrap gap-1.5">
            {flags.split(',').map((flag, idx) => (
              <span
                key={idx}
                className="px-2 py-0.5 rounded bg-blue-500/10 text-blue-300 border border-blue-500/20 font-mono text-[11px]"
              >
                {flag.trim()}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Lexical Diagnostics Audit Trail */}
      {diagnostics && diagnostics.length > 0 && (
        <div className="bg-slate-950/60 border-b border-slate-800 px-4 py-2.5">
          <button
            onClick={() => setShowDiagnostics(!showDiagnostics)}
            className="flex items-center justify-between w-full text-xs font-semibold text-slate-300 hover:text-white transition-colors"
          >
            <div className="flex items-center space-x-2">
              <Terminal className="w-3.5 h-3.5 text-cyan-400" />
              <span>Lexical Audit Trail ({diagnostics.length} AST actions)</span>
            </div>
            {showDiagnostics ? <ChevronUp className="w-3.5 h-3.5 text-slate-400" /> : <ChevronDown className="w-3.5 h-3.5 text-slate-400" />}
          </button>

          {showDiagnostics && (
            <div className="mt-2 space-y-1.5 max-h-[140px] overflow-y-auto">
              {diagnostics.map((diag, idx) => (
                <div
                  key={idx}
                  className="flex items-center space-x-2 text-[11px] font-mono bg-slate-900/80 px-2.5 py-1 rounded border border-slate-800/80 text-slate-300"
                >
                  <span className="px-1.5 py-0.2 rounded bg-slate-800 text-cyan-400 border border-slate-700">
                    L{diag.line}:C{diag.column}
                  </span>
                  <span className="px-1.5 py-0.2 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20">
                    {diag.rule}
                  </span>
                  <span className="text-slate-300 flex-1 truncate">{diag.description}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-slate-800">
        {/* Raw Broken Input */}
        <div className="p-4 flex flex-col">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-rose-400 flex items-center space-x-1">
              <AlertCircle className="w-3.5 h-3.5" />
              <span>Malformed LLM Output</span>
            </span>
            <span className="text-[11px] text-slate-500 font-mono">{original.length} bytes</span>
          </div>
          <pre className="font-mono text-xs text-rose-200/90 bg-slate-950 p-3 rounded-lg border border-rose-950/40 overflow-x-auto whitespace-pre-wrap break-all flex-1 min-h-[160px] max-h-[360px] leading-relaxed">
            {original || '// No content'}
          </pre>
        </div>

        {/* Repaired Clean Output */}
        <div className="p-4 flex flex-col">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium text-emerald-400 flex items-center space-x-1">
              <Check className="w-3.5 h-3.5" />
              <span>Deterministically Healed JSON</span>
            </span>
            <span className="text-[11px] text-slate-500 font-mono">{repaired.length} bytes</span>
          </div>
          <pre className="font-mono text-xs text-emerald-200/90 bg-slate-950 p-3 rounded-lg border border-emerald-950/40 overflow-x-auto whitespace-pre-wrap break-all flex-1 min-h-[160px] max-h-[360px] leading-relaxed">
            {formattedRepaired || '// No repaired output'}
          </pre>
        </div>
      </div>
    </div>
  )
}
