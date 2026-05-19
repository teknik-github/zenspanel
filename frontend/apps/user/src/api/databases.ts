import client from './client'

export const databasesApi = {
  list: () => client.get('/databases'),
  create: (dbName: string, dbUser: string, dbPassword: string) =>
    client.post('/databases', { db_name: dbName, db_user: dbUser, db_password: dbPassword }),
  drop: (id: number) => client.delete(`/databases/${id}`),
  getPHPMyAdminToken: (id: number) => client.get(`/databases/${id}/phpmyadmin`),
  launchPHPMyAdmin: (id: number) => client.get(`/databases/${id}/phpmyadmin/launch`),
}
