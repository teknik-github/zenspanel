import client from './client'

export const backupsApi = {
  list: () => client.get('/backups'),
}
