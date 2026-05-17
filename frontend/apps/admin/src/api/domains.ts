import client from './client'

export const domainsApi = {
  list: (params?: Record<string, unknown>) => client.get('/domains', { params }),
  get: (id: number) => client.get(`/domains/${id}`),
  create: (data: Record<string, unknown>) => client.post('/domains', data),
  update: (id: number, data: Record<string, unknown>) => client.put(`/domains/${id}`, data),
  delete: (id: number) => client.delete(`/domains/${id}`),
  issueSSL: (id: number, data: Record<string, unknown>) => client.post(`/domains/${id}/ssl`, data),
  removeSSL: (id: number) => client.delete(`/domains/${id}/ssl`),
}
