<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { NButton, NCard, NCheckbox, NInput, NProgress, NSpin, useDialog, useMessage } from 'naive-ui'
import { t } from '@/locales'
import { useAuthStore } from '@/store'
import { router } from '@/router'
import type { BackupInfo } from '@/api/system/backup'
import { downloadBackup, fetchBackupInfo, uploadBackup } from '@/api/system/backup'

const ms = useMessage()
const dialog = useDialog()
const authStore = useAuthStore()

const info = ref<BackupInfo | null>(null)
const infoLoading = ref(false)

const includeUploadsInExport = ref(true)
const exporting = ref(false)

const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const restoreUploads = ref(true)
const password = ref('')
const restoring = ref(false)
const restoreProgress = ref(0)

const tableOrder = [
  'user',
  'item_icon_group',
  'item_icon',
  'bookmark',
  'user_config',
  'module_config',
  'file',
  'system_setting',
]

const tableNameMap: Record<string, string> = {
  user: 'apps.backup.tableUser',
  item_icon_group: 'apps.backup.tableItemIconGroup',
  item_icon: 'apps.backup.tableItemIcon',
  bookmark: 'apps.backup.tableBookmark',
  user_config: 'apps.backup.tableUserConfig',
  module_config: 'apps.backup.tableModuleConfig',
  file: 'apps.backup.tableFile',
  system_setting: 'apps.backup.tableSystemSetting',
}

const infoRows = computed(() => {
  if (!info.value)
    return []
  return tableOrder
    .filter(key => key in (info.value?.tables ?? {}))
    .map(key => ({
      key,
      label: tableNameMap[key] ? t(tableNameMap[key]) : key,
      count: info.value?.tables?.[key] ?? 0,
    }))
})

const totalRecords = computed(() => infoRows.value.reduce((sum, row) => sum + row.count, 0))

function formatSize(bytes: number): string {
  if (!bytes)
    return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index++
  }
  return `${index === 0 ? value : value.toFixed(1)} ${units[index]}`
}

async function loadInfo() {
  infoLoading.value = true
  try {
    const res = await fetchBackupInfo()
    if (res.code === 0)
      info.value = res.data
  }
  catch {
    // 概览加载失败不影响导出/恢复主流程
  }
  finally {
    infoLoading.value = false
  }
}

/** 从 Content-Disposition 中解析文件名 */
function parseFileName(headers: unknown, fallback: string): string {
  const raw = headers as Record<string, string | undefined>
  const disposition = raw?.['content-disposition'] ?? ''
  const matched = disposition.match(/filename="?([^";]+)"?/)
  return matched?.[1] || fallback
}

async function handleExport() {
  exporting.value = true
  try {
    const res = await downloadBackup(includeUploadsInExport.value)
    const blob = res.data as Blob

    // 后端在出错时会返回 JSON 而不是 zip
    if (blob.type && blob.type.includes('json')) {
      const text = await blob.text()
      let msg = t('apps.backup.exportFailed')
      try {
        msg = JSON.parse(text)?.msg || msg
      }
      catch {
        // 保持默认错误提示
      }
      ms.error(msg)
      return
    }

    const fileName = parseFileName(
      res.headers,
      `sun-panel-backup-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '')}.zip`,
    )
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = fileName
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    ms.success(t('apps.backup.exportSuccess'))
  }
  catch {
    ms.error(t('apps.backup.exportFailed'))
  }
  finally {
    exporting.value = false
  }
}

function pickFile() {
  fileInput.value?.click()
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) {
    if (!file.name.toLowerCase().endsWith('.zip')) {
      ms.warning(t('apps.backup.onlyZip'))
      selectedFile.value = null
    }
    else {
      selectedFile.value = file
    }
  }
  // 允许重复选择同一个文件
  input.value = ''
}

function handleRestoreClick() {
  if (!selectedFile.value) {
    ms.warning(t('apps.backup.selectFileFirst'))
    return
  }
  if (!password.value) {
    ms.warning(t('apps.backup.passwordRequired'))
    return
  }

  dialog.warning({
    title: t('apps.backup.confirmTitle'),
    content: t('apps.backup.confirmContent'),
    positiveText: t('apps.backup.confirmOk'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => doRestore(),
  })
}

async function doRestore() {
  const file = selectedFile.value
  if (!file)
    return

  restoring.value = true
  restoreProgress.value = 0
  try {
    const res = await uploadBackup(
      file,
      password.value,
      restoreUploads.value,
      percent => (restoreProgress.value = percent),
    )
    const body = res.data
    if (body?.code !== 0) {
      ms.error(body?.msg || t('apps.backup.restoreFailed'))
      return
    }

    const data = body.data
    const summary = t('apps.backup.restoreSummary', {
      records: data?.totalRecords ?? 0,
      uploads: data?.uploadsRestored ?? 0,
    })
    const warningText = data?.warnings?.length
      ? `\n\n${t('apps.backup.restoreWarnings')}\n${data.warnings.join('\n')}`
      : ''

    dialog.success({
      title: t('apps.backup.restoreSuccess'),
      content: () => h('div', { style: 'white-space: pre-line; line-height: 1.7;' }, `${summary}${warningText}`),
      positiveText: t('apps.backup.reloginNow'),
      onPositiveClick: () => {
        authStore.removeToken()
        router.push('/login')
      },
    })

    password.value = ''
    selectedFile.value = null
  }
  catch (error: any) {
    ms.error(error?.response?.data?.msg || error?.msg || t('apps.backup.restoreFailed'))
  }
  finally {
    restoring.value = false
    restoreProgress.value = 0
  }
}

onMounted(() => {
  loadInfo()
})
</script>

<template>
  <div class="bg-slate-200 dark:bg-zinc-900 rounded-[10px] p-[8px] overflow-auto h-full">
    <!-- 数据概览 -->
    <NCard style="border-radius:10px" size="small">
      <div class="flex items-center justify-between">
        <div class="text-slate-500 font-bold">
          {{ t('apps.backup.overview') }}
        </div>
        <NButton size="tiny" :loading="infoLoading" @click="loadInfo">
          {{ t('apps.backup.refresh') }}
        </NButton>
      </div>

      <NSpin :show="infoLoading">
        <div v-if="info" class="mt-[10px]">
          <div class="flex flex-wrap gap-x-[16px] gap-y-[4px] text-xs text-gray-600 dark:text-gray-400">
            <span>{{ t('apps.backup.appVersion') }}: {{ info.appVersion }}</span>
            <span>{{ t('apps.backup.dbDriver') }}: {{ info.dbDriver }}</span>
            <span>{{ t('apps.backup.uploadsDir') }}: {{ info.uploadsDir }}</span>
          </div>

          <div class="mt-[10px] grid grid-cols-2 gap-x-[12px] gap-y-[4px] text-sm">
            <div v-for="row in infoRows" :key="row.key" class="flex justify-between">
              <span class="text-gray-600 dark:text-gray-400">{{ row.label }}</span>
              <span class="font-mono">{{ row.count }}</span>
            </div>
          </div>

          <div class="mt-[8px] text-xs text-gray-600 dark:text-gray-400">
            {{ t('apps.backup.totalRecords', { count: totalRecords }) }} ·
            {{ t('apps.backup.uploadsSummary', { count: info.uploadsCount, size: formatSize(info.uploadsSize) }) }}
          </div>
        </div>
        <div v-else class="mt-[10px] text-xs text-gray-500">
          {{ t('apps.backup.overviewEmpty') }}
        </div>
      </NSpin>
    </NCard>

    <!-- 导出备份 -->
    <NCard style="border-radius:10px" class="mt-[10px]" size="small">
      <div class="text-slate-500 font-bold">
        {{ t('apps.backup.exportTitle') }}
      </div>
      <div class="mt-[8px] text-xs text-gray-600 dark:text-gray-400">
        {{ t('apps.backup.exportHint') }}
      </div>
      <NCheckbox v-model:checked="includeUploadsInExport" class="mt-[8px]">
        {{ t('apps.backup.includeUploads') }}
      </NCheckbox>
      <div class="mt-[10px]">
        <NButton size="small" type="primary" :loading="exporting" @click="handleExport">
          {{ t('apps.backup.exportButton') }}
        </NButton>
      </div>
    </NCard>

    <!-- 恢复数据 -->
    <NCard style="border-radius:10px" class="mt-[10px]" size="small">
      <div class="text-slate-500 font-bold">
        {{ t('apps.backup.restoreTitle') }}
      </div>
      <div class="mt-[8px] text-xs text-red-600 dark:text-red-400">
        {{ t('apps.backup.restoreHint') }}
      </div>

      <input ref="fileInput" type="file" accept=".zip" class="hidden" @change="onFileChange">

      <div class="mt-[10px] flex items-center gap-[8px] flex-wrap">
        <NButton size="small" :disabled="restoring" @click="pickFile">
          {{ t('apps.backup.selectFile') }}
        </NButton>
        <span class="text-xs text-gray-600 dark:text-gray-400">
          {{ selectedFile ? `${selectedFile.name} (${formatSize(selectedFile.size)})` : t('apps.backup.noFileSelected') }}
        </span>
      </div>

      <div class="mt-[10px]">
        <div class="text-xs text-gray-600 dark:text-gray-400 mb-[5px]">
          {{ t('apps.backup.passwordHint') }}
        </div>
        <NInput
          v-model:value="password"
          type="password"
          show-password-on="click"
          :placeholder="t('apps.backup.passwordPlaceholder')"
          :disabled="restoring"
        />
      </div>

      <NCheckbox v-model:checked="restoreUploads" class="mt-[10px]" :disabled="restoring">
        {{ t('apps.backup.restoreUploads') }}
      </NCheckbox>

      <div v-if="restoring" class="mt-[10px]">
        <NProgress type="line" :percentage="restoreProgress" :processing="restoreProgress < 100" />
      </div>

      <div class="mt-[10px]">
        <NButton size="small" type="error" :loading="restoring" :disabled="restoring" @click="handleRestoreClick">
          {{ t('apps.backup.restoreButton') }}
        </NButton>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.text-shadow {
  text-shadow: 0px 0px 5px gray;
}
</style>
