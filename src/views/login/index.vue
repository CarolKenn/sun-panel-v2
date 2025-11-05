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

// const userStore = useUserStore()
const authStore = useAuthStore()
const appStore = useAppStore()
const ms = useMessage()
const loading = ref(false)
const languageValue = ref<Language>(appStore.language)

// const isShowCaptcha = ref<boolean>(false)
// const isShowRegister = ref<boolean>(false)

const form = ref<Login.LoginReqest>({
  username: '',
  password: '',
})

// 导入需要的API函数
import { getList as getGroupList } from '@/api/panel/itemIconGroup'
import { getList as getBookmarksList } from '@/api/panel/bookmark'

const loginPost = async () => {
  loading.value = true
  try {
    const res = await login<Login.LoginResponse>(form.value)
    if (res.code === 0) {
      // 清除所有相关缓存，确保使用新的登录状态数据
      ss.remove('USER_AUTH_INFO_CACHE')
      ss.remove('USER_CONFIG_CACHE')
      ss.remove('GROUP_LIST_CACHE_KEY')
      ss.remove('bookmarksTreeCache')
      
      // 清除所有localStorage中的相关缓存键
      // 对于以特定前缀开头的键，我们需要使用localStorage API直接访问
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i)
        if (key && (key.startsWith('moduleConfig_') || key.startsWith('ITEM_ICON_LIST_CACHE_'))) {
          ss.remove(key)
        }
      }
      
      authStore.setToken(res.data.token)
      authStore.setUserInfo(res.data)
      
      // 登录成功后重新请求所有必要的接口
      // 1. 重新请求分组列表
      const groupListRes = await getGroupList()
      if (groupListRes.code === 0) {
        ss.set('groupListCache', groupListRes.data.list)
        
        // 2. 为每个分组重新请求图标数据
        for (const group of groupListRes.data.list) {
          if (group.id) {
            try {
              const importDynamic = new Function('modulePath', 'return import(modulePath)');
              const { getListByGroupId } = await importDynamic('@/api/panel/itemIcon');
              const iconListRes = await getListByGroupId(group.id);
              if (iconListRes.code === 0) {
                ss.set(`itemIconList_${group.id}`, iconListRes.data.list);
              }
            } catch (importError) {
              console.error('导入getListByGroupId失败:', importError);
            }
          }
        }
      }
      
      // 3. 重新请求书签数据
      const bookmarksRes = await getBookmarksList()
      if (bookmarksRes.code === 0) {
        // 简单处理书签数据，保存到缓存
        ss.set('bookmarksTreeCache', bookmarksRes.data || []);
      }
      
      // 更新本地用户信息
      updateLocalUserInfo(res.data)

      setTimeout(() => {
        ms.success(`Hi ${res.data.name},${t('login.welcomeMessage')}`)
        loading.value = false
        router.push({ path: '/' })
      }, 500)
    }
    else {
      loading.value = false
      // captchaRef.value.refresh()
    }
  }
  catch (error) {
    loading.value = false
    // 请检查网络或者服务器错误
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

        <!-- <NFormItem v-if="isShowCaptcha">
          <div class="w-[120px] h-[34px] mr-[20px] rounded border flex cursor-pointer">
            <Captcha ref="captchaRef" src="/api/captcha/getImage" />
          </div>
          <NInput v-model:value="form.vcode" type="text" placeholder="请输入图像验证码" />
        </NFormItem> -->
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
