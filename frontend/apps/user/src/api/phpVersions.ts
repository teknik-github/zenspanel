import client from './client'

export const phpVersionsApi = {
  listEnabled: () => client.get('/php-versions/enabled'),
}
