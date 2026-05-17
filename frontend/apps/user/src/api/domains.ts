import client from './client'

export const domainsApi = {
  list: () => client.get('/domains'),
  create: (domain: string, phpVersion: string) =>
    client.post('/domains', { domain, php_version: phpVersion }),
  delete: (id: number) => client.delete(`/domains/${id}`),
  updatePHPVersion: (id: number, version: string) =>
    client.put(`/domains/${id}`, { php_version: version }),
}
