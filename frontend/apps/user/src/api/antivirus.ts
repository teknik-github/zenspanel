import client from './client'

export const antivirusApi = {
  status:     () => client.get('/antivirus/status'),
  scan:       (path?: string) => client.post('/antivirus/scan', { path: path || '' }),
  scanStatus: (jobId: string) => client.get(`/antivirus/scan/${jobId}`),
  alerts:     () => client.get('/antivirus/alerts'),
  poll:       () => client.get('/antivirus/poll'),
  watchStart: () => client.post('/antivirus/watch'),
  watchStop:  (watchId: string) => client.delete(`/antivirus/watch/${watchId}`),
}
