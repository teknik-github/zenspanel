import client from './client'

export const phpExtensionsApi = {
  userList: (phpVersion?: string) =>
    client.get('/php-extensions', { params: phpVersion ? { php_version: phpVersion } : {} }),
  userUpdate: (name: string, phpVersion: string, enabled: boolean) =>
    client.put('/php-extensions', { name, php_version: phpVersion, enabled }),
}
