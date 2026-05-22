import client from './client'

export const ipAllowlistApi = {
  list: () => client.get('/admin/ip-allowlist'),
  create: (ip_cidr: string, note: string) => client.post('/admin/ip-allowlist', { ip_cidr, note }),
  delete: (id: number) => client.delete(`/admin/ip-allowlist/${id}`),
}
