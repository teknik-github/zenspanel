import client from './client'

export const packagesApi = {
  list: () => client.get('/packages'),
  get: (id: number) => client.get(`/packages/${id}`),
  create: (data: Record<string, unknown>) => client.post('/packages', data),
  update: (id: number, data: Record<string, unknown>) => client.put(`/packages/${id}`, data),
  delete: (id: number) => client.delete(`/packages/${id}`),
}
