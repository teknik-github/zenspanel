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

export interface UpdateInfo {
  current_sha: string
  latest_sha: string
  behind_by: number
  changelog: string
  current_branch: string
  download_url: string
  release_tag: string
}

export interface UpdateStatus {
  phase: string
  log: string[]
  done: boolean
  error: string
  started_at: string
  finished_at: string
}

export const systemApi = {
  stats:        () => client.get<SystemStats>('/system/stats'),
  checkUpdate:  () => client.get<UpdateInfo>('/system/update/check'),
  runUpdate:    (downloadURL?: string) => client.post<{ started: boolean; error: string }>('/system/update/run', { download_url: downloadURL || '' }),
  updateStatus: () => client.get<UpdateStatus>('/system/update/status'),
}
