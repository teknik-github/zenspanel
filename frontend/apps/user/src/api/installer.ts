import client from './client'

export const installerApi = {
  listApps: () => client.get('/installer/apps'),
  install: (data: {
    app_id: string
    domain_id: number
    db_name?: string
    db_user?: string
    db_pass?: string
    overwrite?: boolean
  }) => client.post('/installer/install', data),
  status: (jobId: string) => client.get(`/installer/status/${jobId}`),
}
