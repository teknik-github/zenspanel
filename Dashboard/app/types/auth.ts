export interface AuthUser {
  id: number
  username: string
  email: string
  role: 'admin' | 'user'
  terminal_enabled: boolean
  backup_enabled: boolean
  package_id: number | null
  php_version: string
  totp_enabled: boolean
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: AuthUser
}

export interface Login2FAResponse {
  requires_2fa: true
  temp_token: string
}

export interface Verify2FARequest {
  temp_token: string
  code: string
}

export interface Recover2FARequest {
  temp_token: string
  recovery_code: string
}
