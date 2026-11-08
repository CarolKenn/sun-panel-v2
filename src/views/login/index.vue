<script setup lang="ts">
import { NButton, NCard, NForm, NFormItem, NGradientText, NInput, NSelect, useMessage } from 'naive-ui'
import { ref } from 'vue'
import { login } from '@/api'
import { useAppStore, useAuthStore } from '@/store'
import { SvgIcon } from '@/components/common'
import { router } from '@/router'
import { t } from '@/locales'
import { languageOptions } from '@/utils/defaultData'
import type { Language } from '@/store/modules/app/helper'
import { ss } from '@/utils/storage'
import {
  CACHE_KEY_BOOKMARKS_TREE,
  CACHE_KEY_GROUP_LIST,
  CACHE_KEY_ITEM_ICON_LIST_PREFIX,
  CACHE_KEY_USER_AUTH_INFO,
  CACHE_KEY_USER_CONFIG,
} from '@/utils/cacheKey'
import { getList as getGroupList } from '@/api/panel/itemIconGroup'
// const userStore = useUserStore()
const authStore = useAuthStore()
const appStore = useAppStore()
const ms = useMessage()
const loading = ref(false)
const languageValue = ref<Language>(appStore.language)
// const isShowRegister = ref<boolean>(false)

const form = ref<Login.LoginReqest>({
  username: '',
  password: '',
})

const loginPost = async () => {
  loading.value = true
  try {
    const res = await login<Login.LoginResponse>(form.value)
    if (res.code === 0) {
      // 清除所有相关缓存，确保使用新的登录状态数据
      ss.remove(CACHE_KEY_USER_AUTH_INFO)
      ss.remove(CACHE_KEY_USER_CONFIG)
      ss.remove(CACHE_KEY_GROUP_LIST)
      ss.remove(CACHE_KEY_BOOKMARKS_TREE)

      // 清除所有localStorage中的相关缓存键
      // 对于以特定前缀开头的键，我们需要使用localStorage API直接访问
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i)
        if (key && (key.startsWith('moduleConfig_') || key.startsWith(CACHE_KEY_ITEM_ICON_LIST_PREFIX))) {
          ss.remove(key)
        }
      }

      authStore.setToken(res.data.token)
      authStore.setUserInfo(res.data)

      // 获取分组列表数据并预写入缓存。
      // 注意：这一步失败只是少了缓存，不能让用户卡在登录页，
      // 所以单独 try/catch，且不 rethrow。
      try {
        const groupListRes = await getGroupList() as any
        if (groupListRes.code === 0 && groupListRes.data)
          ss.set(CACHE_KEY_GROUP_LIST, groupListRes.data.list || [])
      }
      catch (err) {
        console.log('预取分组列表失败，进入首页后会自动重新拉取', err)
      }

      setTimeout(() => {
        ms.success(`Hi ${res.data.name},${t('login.welcomeMessage')}`)
        loading.value = false
        router.push({ path: '/' })
      }, 500)
    }
    else {
      loading.value = false
    }
  }
  catch (error) {
    loading.value = false
    // 网络或服务异常时必须给出提示，否则用户只会看到"点了没反应"
    ms.error(t('common.networkError'))
    console.log(error)
  }
}

function handleSubmit() {
  // 点击登录按钮触发
  loginPost()
}

function handleChangeLanuage(value: Language) {
  languageValue.value = value
  appStore.setLanguage(value)
}
</script>

<template>
  <div class="login-container">
    <NCard class="login-card" style="border-radius: 20px;">
      <div class="mb-5 flex items-center justify-end">
        <div class="mr-2">
          <SvgIcon icon="ion-language" style="width: 20;height: 20;" />
        </div>
        <div class="min-w-[100px]">
          <NSelect v-model:value="languageValue" size="small" :options="languageOptions" @update-value="handleChangeLanuage" />
        </div>
      </div>

      <div class="login-title  ">
        <NGradientText :size="30" type="success" class="!font-bold">
          {{ $t('common.appName') }}
        </NGradientText>
      </div>
      <NForm :model="form" label-width="100px" @keydown.enter="handleSubmit">
        <NFormItem>
          <NInput v-model:value="form.username" :placeholder="$t('login.usernamePlaceholder')">
            <template #prefix>
              <SvgIcon icon="ph:user-bold" />
            </template>
          </NInput>
        </NFormItem>

        <NFormItem>
          <NInput v-model:value="form.password" type="password" :placeholder="$t('login.passwordPlaceholder')">
            <template #prefix>
              <SvgIcon icon="mdi:password-outline" />
            </template>
          </NInput>
        </NFormItem>

        <NFormItem style="margin-top: 10px">
          <NButton type="primary" block :loading="loading" @click="handleSubmit">
            {{ $t('login.loginButton') }}
          </NButton>
        </NFormItem>

        <!-- <div class="flex justify-end">
          <NButton v-if="isShowRegister" quaternary type="info" class="flex" @click="$router.push({ path: '/register' })">
            注册
          </NButton>
          <NButton quaternary type="info" class="flex" @click="$router.push({ path: '/resetPassword' })">
            忘记密码?
          </NButton>
        </div> -->

        <div class="flex justify-center text-slate-300">
          Powered By <a href="https://github.com/75412701/sun-panel-v2" target="_blank" class="ml-[5px] text-slate-500">Sun-Panel</a>
        </div>
      </NForm>
    </NCard>
  </div>
</template>

  <style>
    .login-container {
        padding: 20px;
        display: flex;
        justify-content: center;
        align-items: center;
        height: 100vh;
        background-color: #f2f6ff;
    }

    /* 夜间模式 */
    .dark .login-container{
      background-color: rgb(43, 43, 43);
    }

    @media (min-width: 600px) {
        .login-card {
            width: auto;
            margin: 0px 10px;
        }
        .login-button {
            width: 100%;
        }
    }

    .login-card {
        margin: 20px;
        min-width:400px;
    }

  .login-title{
    text-align: center;
    margin: 20px;
  }
  </style>
