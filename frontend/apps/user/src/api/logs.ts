import client from './client'

export const logsApi = {
  domain: (id: number, type: string, lines: number) =>
    client.get(`/domains/${id}/logs`, { params: { type, lines } }),
}
