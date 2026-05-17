import client from './client'

export const usersApi = {
  list: (params?: Record<string, unknown>) => client.get('/users', { params }),
  get: (id: number) => client.get(`/users/${id}`),
  create: (data: Record<string, unknown>) => client.post('/users', data),
  update: (id: number, data: Record<string, unknown>) => client.put(`/users/${id}`, data),
  delete: (id: number) => client.delete(`/users/${id}`),
  suspend: (id: number) => client.put(`/users/${id}/suspend`),
  unsuspend: (id: number) => client.put(`/users/${id}/unsuspend`),
  changePackage: (id: number, packageId: number) => client.put(`/users/${id}/package`, { package_id: packageId }),
  getUsage: (id: number) => client.get(`/users/${id}/usage`),
}
