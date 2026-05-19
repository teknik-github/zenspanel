import client from './client'

export const phpExtensionsApi = {
  adminList: (phpVersion?: string) =>
    client.get('/admin/php-extensions', { params: phpVersion ? { php_version: phpVersion } : {} }),
  adminUpdate: (id: number, enabled: boolean) =>
    client.put(`/admin/php-extensions/${id}`, { enabled }),
  userList: (phpVersion?: string) =>
    client.get('/php-extensions', { params: phpVersion ? { php_version: phpVersion } : {} }),
  userUpdate: (name: string, phpVersion: string, enabled: boolean) =>
    client.put('/php-extensions', { name, php_version: phpVersion, enabled }),
}
