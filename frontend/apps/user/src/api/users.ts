import client from './client'

export const usersApi = {
  update: (id: number, fields: Record<string, any>) =>
    client.put(`/users/${id}`, fields),
}
