import client from './client'

export const databasesApi = {
  list: (params?: Record<string, unknown>) => client.get('/databases', { params }),
}
