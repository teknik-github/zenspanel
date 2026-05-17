import client from './client'

export const phpVersionsApi = {
  list: () => client.get('/php-versions'),
  listEnabled: () => client.get('/php-versions/enabled'),
  enable: (id: number) => client.put(`/php-versions/${id}/enable`),
  disable: (id: number) => client.put(`/php-versions/${id}/disable`),
}
