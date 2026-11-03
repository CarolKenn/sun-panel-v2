import type { AxiosProgressEvent } from 'axios'
import request from '@/utils/request/axios'
import { get } from '@/utils/request'
import { useAuthStore } from '@/store'

/** 后端统一返回体 */
export interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

/** 当前数据概览 */
export interface BackupInfo {
  appVersion: string
  dbDriver: string
  tables: Record<string, number>
  uploadsDir: string
  uploadsCount: number
  uploadsSize: number
}

/** 备份包元信息 */
export interface BackupManifest {
  format: string
  version: number
  createdAt: string
  appVersion: string
  dbDriver: string
  includeUploads: boolean
  tables?: Record<string, number>
}

/** 恢复结果 */
export interface BackupImportResult {
  tables: Record<string, number>
  totalRecords: number
  uploadsRestored: number
  warnings?: string[]
  manifest?: BackupManifest
  needRelogin: boolean
}

/** 获取当前数据概览 */
export function fetchBackupInfo() {
  return get<BackupInfo>({ url: '/system/backup/info' })
}

/** 导出备份，返回 zip 二进制流 */
export function downloadBackup(includeUploads: boolean) {
  const authStore = useAuthStore()
  return request.post('/system/backup/export', { includeUploads }, {
    responseType: 'blob',
    // 数据量大时导出耗时较长，关闭超时限制
    timeout: 0,
    headers: { token: authStore.token ?? '' },
  })
}

/** 上传备份包并执行恢复 */
export function uploadBackup(
  file: File,
  password: string,
  restoreUploads: boolean,
  onProgress?: (percent: number) => void,
) {
  const authStore = useAuthStore()
  const form = new FormData()
  form.append('file', file)
  form.append('password', password)
  form.append('restoreUploads', String(restoreUploads))

  return request.post<ApiResponse<BackupImportResult>>('/system/backup/import', form, {
    timeout: 0,
    headers: { token: authStore.token ?? '' },
    onUploadProgress: (event: AxiosProgressEvent) => {
      if (onProgress && event.total)
        onProgress(Math.round((event.loaded / event.total) * 100))
    },
  })
}
