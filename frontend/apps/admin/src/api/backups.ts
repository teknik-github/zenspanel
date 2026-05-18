import client from './client'

export const backupsApi = {
  list: () => client.get('/backups'),
  restore: (id: number) => client.post(`/backups/${id}/restore`),
}
