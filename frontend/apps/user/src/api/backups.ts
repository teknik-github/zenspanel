import client from './client'

export const backupsApi = {
  list: () => client.get('/backups'),
  get: (id: number) => client.get(`/backups/${id}`),
  create: (type: 'full' | 'db' | 'files') => client.post('/backups', { type }),
  createDomain: (domainId: number) => client.post(`/domains/${domainId}/backup`),
  download: (id: number) => client.get(`/backups/${id}/download`, { responseType: 'blob' }),
  restore: (id: number) => client.post(`/backups/${id}/restore`),
  delete: (id: number) => client.delete(`/backups/${id}`),
}
