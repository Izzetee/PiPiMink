export { getApiKey, setApiKey } from './client'
export {
  fetchModels,
  toggleModel,
  discoverModels,
  tagModels,
  fetchTagStatus,
  benchmarkModels,
  fetchBenchmarkStatus,
  resetModel,
  deleteModel,
  updateModelReasoning,
  fetchModelBenchmarkResults,
} from './models'
export {
  fetchProviders,
  addProvider,
  updateProvider,
  deleteProvider,
  toggleProvider,
  testProvider,
  updateModelConfigs,
} from './providers'
export {
  fetchBenchmarkTasks,
  fetchLeaderboard,
  fetchAllBenchmarkResults,
  upsertBenchmarkTask,
  deleteBenchmarkTask,
  fetchTaggingPrompts,
  saveTaggingPrompts,
  runBenchmarks,
  testTaggingPrompt,
} from './config'
export {
  fetchSettings,
  patchSettings,
  fetchApiKeys,
  addApiKey,
  editApiKey,
  deleteApiKey,
} from './settings'
export {
  fetchAnalyticsSummary,
  fetchRoutingDecisions,
} from './analytics'
export { fetchAdminStatus } from './status'
export type { AdminStatus } from './status'
export {
  fetchAuthMe,
  logout,
  fetchAuthProviders,
  saveAuthProvider,
  testAuthProvider,
  fetchUsers,
  addLocalUser,
  changeUserRole,
  deleteUser,
  fetchGroups,
  changeGroupRole,
  addRoutingRule,
  removeRoutingRule,
  fetchAuditLog,
} from './auth'
