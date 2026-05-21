import client from './client'

export const redirectsApi = {
  list:   (domainId: number) => client.get(`/domains/${domainId}/redirects`),
  create: (domainId: number, data: { source_path: string; dest_url: string; type?: string; enabled?: boolean }) =>
    client.post(`/domains/${domainId}/redirects`, data),
  update: (domainId: number, rid: number, data: Record<string, any>) =>
    client.put(`/domains/${domainId}/redirects/${rid}`, data),
  delete: (domainId: number, rid: number) =>
    client.delete(`/domains/${domainId}/redirects/${rid}`),
}
