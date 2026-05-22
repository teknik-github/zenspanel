import client from './client'

export const phpVersionsApi = {
  list: () => client.get('/php-versions'),
  listEnabled: () => client.get('/php-versions/enabled'),
  create: (version: string, fpmSocket?: string) => client.post('/php-versions', { version, fpm_socket: fpmSocket }),
  enable: (id: number) => client.put(`/php-versions/${id}/enable`),
  disable: (id: number) => client.put(`/php-versions/${id}/disable`),
  delete: (id: number) => client.delete(`/php-versions/${id}`),
}
