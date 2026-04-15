import { useState, useCallback } from 'react'
import { Shield, Network, Cpu, ArrowRight, Check, Loader2, Sparkles, Copy } from 'lucide-react'
import { patchSettings, addProvider, addApiKey, setApiKey } from '@/api'
import { discoverModels } from '@/api'
import type { ProviderType } from '@/types/provider-management'

interface SetupPageProps {
  onComplete: () => void
}

// Pre-defined provider templates for quick setup
const PROVIDER_TEMPLATES: {
  label: string
  name: string
  type: ProviderType
  baseUrl: string
  apiKeyEnv: string
}[] = [
  { label: 'OpenAI', name: 'openai', type: 'openai-compatible', baseUrl: 'https://api.openai.com', apiKeyEnv: 'OPENAI_API_KEY' },
  { label: 'Anthropic', name: 'anthropic', type: 'anthropic', baseUrl: 'https://api.anthropic.com', apiKeyEnv: 'ANTHROPIC_API_KEY' },
  { label: 'Google Gemini', name: 'gemini', type: 'openai-compatible', baseUrl: 'https://generativelanguage.googleapis.com/v1beta', apiKeyEnv: 'GEMINI_API_KEY' },
  { label: 'OpenRouter', name: 'openrouter', type: 'openai-compatible', baseUrl: 'https://openrouter.ai/api/v1', apiKeyEnv: 'OPENROUTER_API_KEY' },
  { label: 'Local (Ollama)', name: 'local', type: 'openai-compatible', baseUrl: 'http://localhost:11434/v1', apiKeyEnv: '' },
]

function generateKey(): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = 'ppk-'
  for (let i = 0; i < 32; i++) result += chars.charAt(Math.floor(Math.random() * chars.length))
  return result
}

export function SetupPage({ onComplete }: SetupPageProps) {
  const [step, setStep] = useState(0)

  // Step 1: Admin key
  const [adminKey, setAdminKey] = useState('')
  const [keySaving, setKeySaving] = useState(false)
  const [keyError, setKeyError] = useState<string | null>(null)

  // Step 2: Provider
  const [selectedTemplate, setSelectedTemplate] = useState<number | null>(null)
  const [providerApiKey, setProviderApiKey] = useState('')
  const [providerSaving, setProviderSaving] = useState(false)
  const [providerError, setProviderError] = useState<string | null>(null)

  // Step 3: Discover
  const [discovering, setDiscovering] = useState(false)
  const [discoverResult, setDiscoverResult] = useState<{ providers: number; discovered: number } | null>(null)
  const [discoverError, setDiscoverError] = useState<string | null>(null)

  const handleGenerateKey = useCallback(() => {
    setAdminKey(generateKey())
  }, [])

  const handleSaveKey = useCallback(async () => {
    if (!adminKey.trim()) return
    setKeySaving(true)
    setKeyError(null)
    try {
      await patchSettings([{ key: 'ADMIN_API_KEY', category: 'server', previousValue: '', newValue: adminKey.trim() }])
      // Store the key in localStorage so subsequent requests are authenticated
      setApiKey(adminKey.trim())
      setStep(1)
    } catch (err) {
      setKeyError(err instanceof Error ? err.message : 'Failed to save admin key')
    } finally {
      setKeySaving(false)
    }
  }, [adminKey])

  const handleSaveProvider = useCallback(async () => {
    if (selectedTemplate === null) return
    const template = PROVIDER_TEMPLATES[selectedTemplate]
    if (!template) return
    setProviderSaving(true)
    setProviderError(null)
    try {
      // Save the API key to .env first (if provider needs one)
      if (template.apiKeyEnv && providerApiKey.trim()) {
        await addApiKey(template.apiKeyEnv, providerApiKey.trim())
      }
      // Create the provider
      await addProvider({
        name: template.name,
        type: template.type,
        baseUrl: template.baseUrl,
        apiKeyEnv: template.apiKeyEnv,
        timeout: '2m',
        rateLimitSeconds: null,
        enabled: true,
      })
      setStep(2)
    } catch (err) {
      setProviderError(err instanceof Error ? err.message : 'Failed to add provider')
    } finally {
      setProviderSaving(false)
    }
  }, [selectedTemplate, providerApiKey])

  const handleDiscover = useCallback(async () => {
    setDiscovering(true)
    setDiscoverError(null)
    try {
      const result = await discoverModels()
      setDiscoverResult(result)
    } catch (err) {
      setDiscoverError(err instanceof Error ? err.message : 'Failed to discover models')
    } finally {
      setDiscovering(false)
    }
  }, [])

  const steps = [
    { label: 'Secure', icon: Shield },
    { label: 'Connect', icon: Network },
    { label: 'Discover', icon: Cpu },
  ]

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 flex items-center justify-center p-4">
      <div className="w-full max-w-lg">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 mb-3">
            <Sparkles className="w-6 h-6 text-indigo-500" strokeWidth={1.75} />
            <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">
              Welcome to PiPiMink
            </h1>
          </div>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            Route every prompt to the best model — for you specifically.
          </p>
        </div>

        {/* Step indicators */}
        <div className="flex items-center justify-center gap-2 mb-8">
          {steps.map((s, i) => {
            const Icon = s.icon
            const done = i < step
            const active = i === step
            return (
              <div key={s.label} className="flex items-center gap-2">
                <div
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium transition-colors ${
                    done
                      ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400'
                      : active
                        ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-400'
                        : 'bg-slate-100 text-slate-400 dark:bg-slate-800 dark:text-slate-500'
                  }`}
                >
                  {done ? <Check className="w-3 h-3" /> : <Icon className="w-3 h-3" />}
                  {s.label}
                </div>
                {i < steps.length - 1 && (
                  <ArrowRight className="w-3 h-3 text-slate-300 dark:text-slate-600" />
                )}
              </div>
            )
          })}
        </div>

        {/* Card */}
        <div className="rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800 shadow-sm p-6">

          {/* Step 1: Admin Key */}
          {step === 0 && (
            <div>
              <h2 className="text-lg font-medium text-slate-900 dark:text-slate-100 mb-1">
                Set an admin API key
              </h2>
              <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
                This key protects the Console and admin APIs. Store it somewhere safe.
              </p>

              <div className="space-y-3">
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={adminKey}
                    onChange={(e) => setAdminKey(e.target.value)}
                    placeholder="Enter or generate a key..."
                    className="flex-1 px-3 py-2 text-sm rounded-lg border border-slate-200 bg-slate-50 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 font-mono"
                  />
                  <button
                    onClick={handleGenerateKey}
                    className="px-3 py-2 text-sm rounded-lg border border-slate-200 hover:bg-slate-50 dark:border-slate-600 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 transition-colors"
                    title="Generate random key"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                </div>

                {keyError && (
                  <p className="text-xs text-red-600 dark:text-red-400">{keyError}</p>
                )}

                <div className="flex items-center justify-between pt-2">
                  <button
                    onClick={() => setStep(1)}
                    className="text-sm text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
                  >
                    Skip for now
                  </button>
                  <button
                    onClick={handleSaveKey}
                    disabled={!adminKey.trim() || keySaving}
                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {keySaving ? <Loader2 className="w-4 h-4 animate-spin" /> : <ArrowRight className="w-4 h-4" />}
                    Continue
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Step 2: Add Provider */}
          {step === 1 && (
            <div>
              <h2 className="text-lg font-medium text-slate-900 dark:text-slate-100 mb-1">
                Add a provider
              </h2>
              <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
                Connect an LLM provider. You can add more later in the Providers page.
              </p>

              <div className="space-y-3">
                {/* Template buttons */}
                <div className="grid grid-cols-2 gap-2">
                  {PROVIDER_TEMPLATES.map((t, i) => (
                    <button
                      key={t.name}
                      onClick={() => setSelectedTemplate(i)}
                      className={`px-3 py-2 text-sm rounded-lg border text-left transition-colors ${
                        selectedTemplate === i
                          ? 'border-indigo-500 bg-indigo-50 text-indigo-700 dark:border-indigo-400 dark:bg-indigo-900/30 dark:text-indigo-300'
                          : 'border-slate-200 hover:bg-slate-50 text-slate-700 dark:border-slate-600 dark:hover:bg-slate-700 dark:text-slate-300'
                      }`}
                    >
                      {t.label}
                    </button>
                  ))}
                </div>

                {/* API key input (if template selected and needs a key) */}
                {selectedTemplate !== null && PROVIDER_TEMPLATES[selectedTemplate]?.apiKeyEnv && (
                  <div>
                    <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                      {PROVIDER_TEMPLATES[selectedTemplate]?.apiKeyEnv}
                    </label>
                    <input
                      type="password"
                      value={providerApiKey}
                      onChange={(e) => setProviderApiKey(e.target.value)}
                      placeholder="Paste your API key..."
                      className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 bg-slate-50 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 font-mono"
                    />
                  </div>
                )}

                {providerError && (
                  <p className="text-xs text-red-600 dark:text-red-400">{providerError}</p>
                )}

                <div className="flex items-center justify-between pt-2">
                  <button
                    onClick={() => setStep(2)}
                    className="text-sm text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
                  >
                    Skip for now
                  </button>
                  <button
                    onClick={handleSaveProvider}
                    disabled={selectedTemplate === null || providerSaving}
                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {providerSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : <ArrowRight className="w-4 h-4" />}
                    Continue
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Step 3: Discover Models */}
          {step === 2 && (
            <div>
              <h2 className="text-lg font-medium text-slate-900 dark:text-slate-100 mb-1">
                Discover models
              </h2>
              <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
                Query your providers to find available models. This may take a moment.
              </p>

              <div className="space-y-3">
                {!discoverResult && !discovering && (
                  <button
                    onClick={handleDiscover}
                    className="w-full flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 transition-colors"
                  >
                    <Cpu className="w-4 h-4" />
                    Discover Models
                  </button>
                )}

                {discovering && (
                  <div className="flex items-center justify-center gap-2 py-6 text-sm text-slate-500 dark:text-slate-400">
                    <Loader2 className="w-5 h-5 animate-spin text-indigo-500" />
                    Querying providers...
                  </div>
                )}

                {discoverResult && (
                  <div className="rounded-lg border border-emerald-200 bg-emerald-50/60 dark:border-emerald-500/20 dark:bg-emerald-950/30 px-4 py-3">
                    <div className="flex items-center gap-2">
                      <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                      <p className="text-sm font-medium text-emerald-800 dark:text-emerald-300">
                        Found {discoverResult.discovered} model{discoverResult.discovered !== 1 ? 's' : ''} from{' '}
                        {discoverResult.providers} provider{discoverResult.providers !== 1 ? 's' : ''}
                      </p>
                    </div>
                  </div>
                )}

                {discoverError && (
                  <p className="text-xs text-red-600 dark:text-red-400">{discoverError}</p>
                )}

                <div className="flex items-center justify-between pt-2">
                  <button
                    onClick={onComplete}
                    className="text-sm text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
                  >
                    {discoverResult ? '' : 'Skip for now'}
                  </button>
                  <button
                    onClick={onComplete}
                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 transition-colors"
                  >
                    Go to Console
                    <ArrowRight className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-slate-400 dark:text-slate-500 mt-4">
          You can change all of these settings later in the Console.
        </p>
      </div>
    </div>
  )
}
