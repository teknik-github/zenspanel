import client from './client'

export const sslApi = {
  issueLetsEncrypt: (domainId: number) =>
    client.post(`/domains/${domainId}/ssl`, { type: 'letsencrypt' }),
  uploadCustom: (domainId: number, cert: string, key: string) =>
    client.post(`/domains/${domainId}/ssl`, { type: 'custom', cert_pem: cert, key_pem: key }),
  remove: (domainId: number) => client.delete(`/domains/${domainId}/ssl`),
}
