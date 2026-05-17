import client from './client'

export const usageApi = {
  get: (userId: number) => client.get(`/users/${userId}/usage`),
}
