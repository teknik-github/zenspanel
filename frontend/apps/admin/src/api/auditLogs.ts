import client from './client'

export const auditLogsApi = {
  list: (params?: Record<string, unknown>) => client.get('/audit-logs', { params }),
}
