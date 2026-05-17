import client from './client'

export const terminalApi = {
  getToken: () => client.post<{ token: string }>('/terminal/token'),
}
