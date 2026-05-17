import client from './client'

export const apiKeysApi = {
  list: () => client.get('/api-keys'),
  create: (data: Record<string, unknown>) => client.post('/api-keys', data),
  revoke: (id: number) => client.delete(`/api-keys/${id}`),
}
