import client from './client'

export const packagesApi = {
  get: (id: number) => client.get(`/packages/${id}`),
}
