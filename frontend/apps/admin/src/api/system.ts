import client from './client'

export interface SystemStats {
  users: { total: number; active: number; suspended: number }
  domains: { total: number; active: number }
  databases: { total: number }
  cpu_percent: number
  ram_used: number
  ram_total: number
  services: { nginx: string; mysql: string; redis: string }
  uptime_seconds: number
}

export const systemApi = {
  stats: () => client.get<SystemStats>('/system/stats'),
}
