import React from 'react'
import { Activity, ShieldCheck, Cpu, Code2, Database } from 'lucide-react'

interface HeaderProps {
  activeTab: 'dashboard' | 'playground' | 'schemas'
  setActiveTab: (tab: 'dashboard' | 'playground' | 'schemas') => void
  isConnected: boolean
}

export const Header: React.FC<HeaderProps> = ({ activeTab, setActiveTab, isConnected }) => {
  return (
    <header className="border-b border-slate-800 bg-slate-900/60 backdrop-blur sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div className="h-9 w-9 rounded-lg bg-blue-600/20 border border-blue-500/30 flex items-center justify-center text-blue-400">
            <ShieldCheck className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <span className="font-bold text-lg tracking-tight text-white font-mono">schemadoctor</span>
              <span className="text-xs px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20 font-medium">
                v1.0.0
              </span>
            </div>
            <p className="text-xs text-slate-400">Zero-Latency JSON Self-Healing & AST Repair Proxy</p>
          </div>
        </div>

        {/* Navigation tabs */}
        <nav className="flex space-x-1 bg-slate-950 p-1 rounded-lg border border-slate-800">
          <button
            onClick={() => setActiveTab('dashboard')}
            className={`flex items-center space-x-2 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
              activeTab === 'dashboard'
                ? 'bg-blue-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
            }`}
          >
            <Activity className="w-3.5 h-3.5" />
            <span>Live Monitor</span>
          </button>
          <button
            onClick={() => setActiveTab('playground')}
            className={`flex items-center space-x-2 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
              activeTab === 'playground'
                ? 'bg-blue-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
            }`}
          >
            <Code2 className="w-3.5 h-3.5" />
            <span>Interactive Playground</span>
          </button>
          <button
            onClick={() => setActiveTab('schemas')}
            className={`flex items-center space-x-2 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
              activeTab === 'schemas'
                ? 'bg-blue-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
            }`}
          >
            <Database className="w-3.5 h-3.5" />
            <span>Schemas</span>
          </button>
        </nav>

        {/* Live SSE Status */}
        <div className="flex items-center space-x-4">
          <div className="flex items-center space-x-2 text-xs">
            <span className="relative flex h-2 w-2">
              <span className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${
                isConnected ? 'bg-emerald-400' : 'bg-amber-400'
              }`} />
              <span className={`relative inline-flex rounded-full h-2 w-2 ${
                isConnected ? 'bg-emerald-500' : 'bg-amber-500'
              }`} />
            </span>
            <span className="text-slate-400 font-mono">
              {isConnected ? 'STREAM CONNECTED' : 'POLLING'}
            </span>
          </div>

          <div className="hidden sm:flex items-center space-x-1.5 px-2.5 py-1 rounded bg-slate-800/80 border border-slate-700 text-xs text-slate-300">
            <Cpu className="w-3.5 h-3.5 text-blue-400" />
            <span>Wire latency &lt; 1ms</span>
          </div>
        </div>
      </div>
    </header>
  )
}
