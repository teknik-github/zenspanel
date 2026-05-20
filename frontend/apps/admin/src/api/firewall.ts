import client from './client'

export const firewallApi = {
  listBlocked: () => client.get('/admin/firewall/blocked'),
  block: (ip: string, reason?: string) => client.post('/admin/firewall/block', { ip, reason }),
  unblock: (ip: string) => client.post('/admin/firewall/unblock', { ip }),
  listJails: () => client.get('/admin/firewall/fail2ban/jails'),
  setJail: (name: string, enabled: boolean) =>
    client.put(`/admin/firewall/fail2ban/jails/${name}`, { enabled }),
}
