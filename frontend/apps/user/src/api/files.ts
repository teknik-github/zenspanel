import client from './client'

export interface FileEntry {
  name: string
  size: number
  is_dir: boolean
  mod_time: number
  mode: string
}

export const filesApi = {
  list: (path: string) => client.get<{ entries: FileEntry[] }>('/files', { params: { path } }),
  read: (path: string) => client.get<{ content: string }>('/files/content', { params: { path } }),
  write: (path: string, content: string) => client.post('/files/content', { path, content }),
  mkdir: (path: string) => client.post('/files/mkdir', { path }),
  rename: (oldPath: string, newPath: string) => client.put('/files/rename', { old_path: oldPath, new_path: newPath }),
  delete: (path: string) => client.delete('/files', { params: { path } }),
  upload: (destDir: string, file: File, onProgress?: (pct: number) => void) => {
    const fd = new FormData()
    fd.append('path', destDir)
    fd.append('file', file)
    return client.post('/files/upload', fd, {
      onUploadProgress: e => {
        if (onProgress && e.total) onProgress(Math.round((e.loaded * 100) / e.total))
      },
    })
  },
}

