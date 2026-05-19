import client from './client'

export const subdomainsApi = {
  // listByParent fetches the subdomains belonging to a single parent
  // domain. The handler also accepts no parent_id (returns all of the
  // current user's subdomains) — useful for the SSL Manager page where
  // we want a flat list rather than per-parent grouping.
  listByParent: (parentId: number) =>
    client.get(`/subdomains?parent_id=${parentId}`),
  list: () => client.get('/subdomains'),
  get: (id: number) => client.get(`/subdomains/${id}`),
  create: (parentDomainId: number, subdomain: string, phpVersion: string, docRoot?: string) =>
    client.post('/subdomains', {
      parent_domain_id: parentDomainId,
      subdomain,
      php_version: phpVersion,
      doc_root: docRoot,
    }),
  updatePHPVersion: (id: number, version: string) =>
    client.put(`/subdomains/${id}`, { php_version: version }),
  delete: (id: number) => client.delete(`/subdomains/${id}`),

  // SSL — same shape as ssl.ts but routed through /subdomains/:id/ssl
  // so the handler picks the subdomain row instead of a domain row.
  issueLetsEncrypt: (id: number) =>
    client.post(`/subdomains/${id}/ssl`, { type: 'letsencrypt' }),
  uploadCustomSSL: (id: number, certPEM: string, keyPEM: string) =>
    client.post(`/subdomains/${id}/ssl`, {
      type: 'custom',
      cert_pem: certPEM,
      key_pem: keyPEM,
    }),
  removeSSL: (id: number) => client.delete(`/subdomains/${id}/ssl`),
}
