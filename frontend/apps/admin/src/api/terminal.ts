import client from './client'

export const terminalApi = {
  adminToken: (username?: string) =>
    client.post('/admin/terminal/token', { username: username || '' }),
}
