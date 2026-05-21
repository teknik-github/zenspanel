import client from './client'

export const hotlinkApi = {
  get: (domainId: number) => client.get(`/domains/${domainId}/hotlink`),
  set: (domainId: number, enabled: boolean, allowedDomains: string[]) =>
    client.put(`/domains/${domainId}/hotlink`, { enabled, allowed_domains: allowedDomains }),
}
