import client from './client'

export const cronJobsApi = {
  list: () => client.get('/cron-jobs'),
  create: (data: { expression: string; command: string; enabled?: boolean }) =>
    client.post('/cron-jobs', data),
  update: (id: number, data: Record<string, unknown>) =>
    client.put(`/cron-jobs/${id}`, data),
  delete: (id: number) => client.delete(`/cron-jobs/${id}`),
}
