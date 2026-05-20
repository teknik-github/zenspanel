import client from './client'

export const authApi = {
  login: (username: string, password: string) =>
    client.post('/auth/login', { username, password }),
  me: () => client.get('/auth/me'),
  twofa: {
    setup:   ()                                    => client.post('/auth/2fa/setup'),
    confirm: (code: string)                        => client.post('/auth/2fa/confirm', { code }),
    disable: (code: string)                        => client.delete('/auth/2fa', { data: { code } }),
    verify:  (temp_token: string, code: string)    => client.post('/auth/2fa/verify', { temp_token, code }),
    recover: (temp_token: string, recovery_code: string) =>
      client.post('/auth/2fa/recover', { temp_token, recovery_code }),
  },
}
