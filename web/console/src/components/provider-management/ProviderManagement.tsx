import { useState, useMemo } from 'react'
import type {
  ProviderManagementProps,
  Provider,
  ProviderType,
} from '@/types/provider-management'
import { ProviderList } from './ProviderList'
import { ProviderDetail } from './ProviderDetail'
import { ProviderModal } from './ProviderModal'
import { ConfirmDialog } from './ConfirmDialog'
import {
  Plus,
  Server,
} from 'lucide-react'

export function ProviderManagement({
  providers,
  onAddProvider,
  onEditProvider,
  onDeleteProvider,
  onDuplicateProvider,
  onToggleProvider,
  onToggleModelConfigs,
  onTestConnection,
  onAddModelConfig,
  onEditModelConfig,
  onToggleModelConfig,
  onDeleteModelConfig,
}: ProviderManagementProps) {
  const [selectedId, setSelectedId] = useState<string | null>(
    providers[0]?.id ?? null
  )
  const [modalOpen, setModalOpen] = useState(false)
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Provider | null>(null)

  const selectedProvider = useMemo(
    () => providers.find((p) => p.id === selectedId) ?? null,
    [providers, selectedId]
  )

  const handleAdd = () => {
    setEditingProvider(null)
    setModalOpen(true)
  }

  const handleEdit = (provider: Provider) => {
    setEditingProvider(provider)
    setModalOpen(true)
  }

  const handleSave = (data: {
    name: string
    type: ProviderType
    baseUrl: string
    apiKeyEnv: string
    timeout: string
    rateLimitSeconds: number | null
    enabled: boolean
  }) => {
    if (editingProvider) {
      onEditProvider?.(editingProvider.id, data)
    } else {
      onAddProvider?.({ ...data, hasModelConfigs: false })
    }
    setModalOpen(false)
    setEditingProvider(null)
  }

  const handleDelete = (provider: Provider) => {
    setDeleteTarget(provider)
  }

  const confirmDelete = () => {
    if (deleteTarget) {
      onDeleteProvider?.(deleteTarget.id)
      if (selectedId === deleteTarget.id) {
        setSelectedId(providers.find((p) => p.id !== deleteTarget.id)?.id ?? null)
      }
    }
    setDeleteTarget(null)
  }

  return (
    <div className="h-full flex flex-col md:flex-row">
      {/* Sidebar — provider list */}
      <div className="w-full md:w-72 lg:w-80 shrink-0 border-b md:border-b-0 md:border-r border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 flex flex-col">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700/50 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
            Providers
          </h2>
          <button
            onClick={handleAdd}
            className="inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm"
          >
            <Plus className="w-3.5 h-3.5" strokeWidth={2} />
            Add
          </button>
        </div>

        <div className="flex-1 overflow-y-auto">
          {providers.length === 0 ? (
            <div className="py-16 text-center px-4">
              <Server
                className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-3"
                strokeWidth={1}
              />
              <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
                No providers configured
              </p>
              <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
                Add a provider to get started
              </p>
            </div>
          ) : (
            <ProviderList
              providers={providers}
              selectedId={selectedId}
              onSelect={setSelectedId}
            />
          )}
        </div>
      </div>

      {/* Detail panel */}
      <div className="flex-1 overflow-y-auto bg-slate-50 dark:bg-slate-900">
        {selectedProvider ? (
          <ProviderDetail
            provider={selectedProvider}
            onEdit={() => handleEdit(selectedProvider)}
            onDelete={() => handleDelete(selectedProvider)}
            onDuplicate={() => onDuplicateProvider?.(selectedProvider.id)}
            onToggle={(enabled) =>
              onToggleProvider?.(selectedProvider.id, enabled)
            }
            onToggleModelConfigs={(hasModelConfigs) =>
              onToggleModelConfigs?.(selectedProvider.id, hasModelConfigs)
            }
            onTestConnection={() => onTestConnection?.(selectedProvider.id)}
            onAddModelConfig={(config) =>
              onAddModelConfig?.(selectedProvider.id, config)
            }
            onEditModelConfig={(configId, updates) =>
              onEditModelConfig?.(selectedProvider.id, configId, updates)
            }
            onToggleModelConfig={(configId, enabled) =>
              onToggleModelConfig?.(selectedProvider.id, configId, enabled)
            }
            onDeleteModelConfig={(configId) =>
              onDeleteModelConfig?.(selectedProvider.id, configId)
            }
          />
        ) : (
          <div className="h-full flex items-center justify-center">
            <div className="text-center">
              <Server
                className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3"
                strokeWidth={1}
              />
              <p className="text-sm text-slate-500 dark:text-slate-400">
                Select a provider to view details
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Add/Edit modal */}
      {modalOpen && (
        <ProviderModal
          provider={editingProvider}
          onSave={handleSave}
          onClose={() => {
            setModalOpen(false)
            setEditingProvider(null)
          }}
        />
      )}

      {/* Delete confirmation */}
      {deleteTarget && (
        <ConfirmDialog
          title="Delete provider"
          message={`Are you sure you want to delete "${deleteTarget.name}"? This action cannot be undone.`}
          confirmLabel="Delete"
          variant="danger"
          onConfirm={confirmDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
