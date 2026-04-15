import { useState } from 'react'
import { Shield } from 'lucide-react'
import { setApiKey } from '@/api/client'

interface LoginPageProps {
  oauthEnabled: boolean
}

export function LoginPage({ oauthEnabled }: LoginPageProps) {
  const [apiKeyValue, setApiKeyValue] = useState('')

  function handleApiKeySubmit(e: React.FormEvent) {
    e.preventDefault()
    if (apiKeyValue.trim()) {
      setApiKey(apiKeyValue.trim())
      window.location.href = '/console/models'
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900 p-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="w-12 h-12 rounded-xl bg-indigo-600 flex items-center justify-center mx-auto mb-3">
            <Shield className="w-6 h-6 text-white" strokeWidth={1.75} />
          </div>
          <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">
            PiPiMink Console
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Sign in to continue
          </p>
        </div>

        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm p-6 space-y-4">
          {oauthEnabled && (
            <>
              <a
                href="/auth/login"
                className="w-full flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium rounded-lg bg-indigo-600 dark:bg-indigo-500 text-white hover:bg-indigo-700 dark:hover:bg-indigo-600 transition-colors"
              >
                <Shield className="w-4 h-4" />
                Sign in with Authentik
              </a>

              <div className="flex items-center gap-3">
                <div className="flex-1 h-px bg-slate-200 dark:bg-slate-700" />
                <span className="text-xs text-slate-400 dark:text-slate-500">or</span>
                <div className="flex-1 h-px bg-slate-200 dark:bg-slate-700" />
              </div>
            </>
          )}

          <form onSubmit={handleApiKeySubmit} className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
                Admin API Key
              </label>
              <input
                type="password"
                value={apiKeyValue}
                onChange={(e) => setApiKeyValue(e.target.value)}
                placeholder="Enter your API key"
                className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-500"
              />
            </div>
            <button
              type="submit"
              disabled={!apiKeyValue.trim()}
              className="w-full px-4 py-2.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-600 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Sign in with API Key
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
