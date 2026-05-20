import client from './client'

export const backupTargetsApi = {
  list:   () => client.get('/admin/backup-targets'),
  create: (data: Record<string, any>) => client.post('/admin/backup-targets', data),
  update: (id: number, data: Record<string, any>) => client.put(`/admin/backup-targets/${id}`, data),
  delete: (id: number) => client.delete(`/admin/backup-targets/${id}`),
  test:   (id: number) => client.post(`/admin/backup-targets/${id}/test`),
}
