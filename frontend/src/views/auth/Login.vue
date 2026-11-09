<template>
  <div class="login-page">
    <!-- Askademic 开场动画「破土问学」:每会话一次,点击跳过 -->
    <BrandIntro />

    <!-- 语言切换（功能保留自旧版，位置固定右上） -->
    <div class="lang-switch language-switch">
      <button @click="toggleLanguageMenu" class="header-link" :title="currentLangOption?.label">
        <span class="lang-flag-icon">{{ currentLangOption?.flag }}</span>
        <span class="link-text">{{ currentLangOption?.shortLabel }}</span>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"
          stroke-linecap="round">
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>
      <div v-if="showLanguageMenu" class="language-dropdown">
        <div v-for="lang in languageOptions" :key="lang.value" @click="selectLanguage(lang.value)"
          class="language-option" :class="{ active: currentLanguage === lang.value }">
          <span class="lang-flag">{{ lang.flag }}</span>
          <span class="lang-label">{{ lang.label }}</span>
          <span v-if="currentLanguage === lang.value" class="check-icon">✓</span>
        </div>
      </div>
    </div>

    <div class="stage">
      <!-- 右列：表单（DOM 在前，手机端自然先渲染表单） -->
      <div class="formcol">
        <div class="brand-block">
          <BrandWordmark :height="30" class="brand-logo" />
          <h1 class="sr-only-brand">Askademic</h1>
          <p class="brand-sub">{{ $t('auth.subtitle') }}</p>
        </div>

        <!-- 登录卡（逻辑与旧版一致） -->
        <div class="form-card" v-if="!isRegisterMode">
          <!-- 登录已有账号时保留邀请上下文。 -->
          <div v-if="inviteLookup" class="invite-banner">
            <t-icon name="link" class="invite-banner__icon" />
            <div class="invite-banner__text">
              <div class="invite-banner__title">
                {{ $t('inviteRegister.bannerTitle', { tenant: inviteLookup.tenant_name || '' }) }}
              </div>
              <div class="invite-banner__hint">
                {{ $t('inviteRegister.bannerHintLogin') }}
              </div>
            </div>
          </div>
          <div v-else-if="inviteLookupError" class="invite-banner invite-banner--error">
            {{ inviteLookupError }}
          </div>
          <div class="form-header">
            <h2 class="form-title">{{ $t('auth.login') }}</h2>
            <p v-if="registrationEnabled" class="form-hint">{{ $t('auth.loginHint') }}</p>
          </div>

          <div class="form-content">
            <t-form ref="formRef" :data="formData" :rules="formRules" @submit="handleLogin" layout="vertical"
              label-align="top">
              <t-form-item :label="$t('auth.email')" name="email">
                <t-input v-model="formData.email" :placeholder="$t('auth.emailPlaceholder')" type="text"
                  autocomplete="email" size="large" :disabled="loading" />
              </t-form-item>

              <t-form-item :label="$t('auth.password')" name="password">
                <t-input v-model="formData.password" :placeholder="$t('auth.passwordPlaceholder')" type="password"
                  autocomplete="current-password" size="large" :disabled="loading" @enter="handleLogin" />
              </t-form-item>

              <t-button type="submit" theme="primary" size="large" block :loading="loading" class="submit-button">
                {{ loading ? $t('auth.loggingIn') : $t('auth.login') }}
              </t-button>

              <div class="register-cta" v-if="registrationEnabled">
                <div class="register-cta__divider">
                  <span>{{ $t('auth.firstTime') }}</span>
                </div>
                <t-button theme="default" variant="outline" size="large" block class="register-cta__button"
                  :disabled="loading" @click="toggleMode">
                  {{ $t('auth.createAccount') }}
                </t-button>
              </div>

              <div v-if="oidcEnabled" class="oidc-divider">
                <span>{{ $t('auth.orContinueWith') }}</span>
              </div>

              <t-button v-if="oidcEnabled" theme="default" size="large" block :loading="oidcLoading" :disabled="loading"
                class="oidc-button" @click="handleOIDCLogin">
                {{ oidcLoading ? $t('auth.redirectingToOIDC') : oidcLoginText }}
              </t-button>
            </t-form>
          </div>
        </div>

        <!-- 注册卡（逻辑与旧版一致） -->
        <div class="form-card" v-if="isRegisterMode && registrationEnabled">
          <div v-if="inviteLookup" class="invite-banner">
            <t-icon name="link" class="invite-banner__icon" />
            <div class="invite-banner__text">
              <div class="invite-banner__title">
                {{ $t('inviteRegister.bannerTitle', { tenant: inviteLookup.tenant_name || '' }) }}
              </div>
              <div class="invite-banner__hint">
                {{ $t('inviteRegister.bannerHint') }}
              </div>
            </div>
          </div>
          <div v-else-if="inviteLookupError" class="invite-banner invite-banner--error">
            {{ inviteLookupError }}
          </div>
          <div class="form-header">
            <h2 class="form-title">{{ $t('auth.createAccount') }}</h2>
            <p class="form-subtitle">{{ $t('auth.registerSubtitle') }}</p>
          </div>

          <div class="form-content">
            <t-form ref="registerFormRef" :data="registerData" :rules="registerRules" @submit="handleRegister"
              layout="vertical" label-align="top">
              <t-form-item :label="$t('auth.username')" name="username">
                <t-input v-model="registerData.username" :placeholder="$t('auth.usernamePlaceholder')" size="large"
                  :disabled="loading" />
              </t-form-item>

              <t-form-item :label="$t('auth.email')" name="email">
                <t-input v-model="registerData.email" :placeholder="$t('auth.emailPlaceholder')" type="text"
                  autocomplete="email" size="large" :disabled="loading" />
              </t-form-item>

              <t-form-item :label="$t('auth.password')" name="password">
                <t-input v-model="registerData.password" :placeholder="$t('auth.passwordPlaceholder')" type="password"
                  autocomplete="new-password" size="large" :disabled="loading" />
              </t-form-item>

              <t-form-item :label="$t('auth.confirmPassword')" name="confirmPassword">
                <t-input v-model="registerData.confirmPassword" :placeholder="$t('auth.confirmPasswordPlaceholder')"
                  type="password" autocomplete="new-password" size="large" :disabled="loading"
                  @enter="handleRegister" />
              </t-form-item>

              <t-button type="submit" theme="primary" size="large" block :loading="loading" class="submit-button">
                {{ loading ? $t('auth.registering') : $t('auth.register') }}
              </t-button>
            </t-form>

            <div class="form-footer">
              <span>{{ $t('auth.haveAccount') }}</span>
              <a href="#" @click.prevent="toggleMode" class="link-button">
                {{ $t('auth.backToLogin') }}
              </a>
            </div>
          </div>
        </div>

        <p class="foot">
          <b>Askademic</b> · {{ $t('auth.subtitle') }}
        </p>
      </div>

      <!-- 左列：产品切片（data-ui）。能力随真实数据逐步接入：
           第一期后端落地后，这里升级为真实问候 + 今日课表 + 带出处的回答切片；
           当前只陈述产品定义（来自 docs/superpowers/specs），不展示伪造数据。 -->
      <div class="sheet" data-ui aria-label="Askademic 产品预览">
        <div class="mini-brand">
          <BrandWordmark :height="18" />
        </div>
        <div class="statement-block">
          <p class="statement">制度问题有据可查，<br>个人事务即问即答。</p>
          <p class="statement-sub">面向高校教务场景的智能问答助手：制度、课表、个人事务，问一句就答，回答附知识库出处。</p>
        </div>
        <div class="cap-panel">
          <div class="cap-row">
            <span><span class="cap-title">制度问答</span><br><span class="cap-sub">学籍、选课、培养方案、毕业要求，回答附知识库出处</span></span>
          </div>
          <div class="cap-row">
            <span><span class="cap-title">个人事务</span><br><span class="cap-sub">学分、绩点、课表冲突，按你的角色权限即问即答</span></span>
          </div>
          <div class="cap-row">
            <span><span class="cap-title">班级空间</span><br><span class="cap-sub">制度文库、班级资料、培养方案库，按空间共享</span></span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, nextTick, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoleLabel } from '@/composables/useRoleLabel'
import { notifyLoginSuccess } from '@/utils/loginNotify'
import { canRegister } from './registrationPolicy'
import { newPasswordRules } from '@/utils/passwordPolicy'
import {
  login,
  register,
  getOIDCAuthorizationURL,
  getOIDCConfig,
  autoSetup,
  getAuthConfig,
  userInfoFromApi,
  getInvitationByToken,
  registerByInvite,
  type InviteLookup,
} from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import BrandWordmark from '@/components/BrandWordmark.vue'
import { useI18n } from 'vue-i18n'
import BrandIntro from '@/components/BrandIntro.vue'
import { getApiBaseUrl } from '@/utils/api-base'
import { BRAND } from '@/config/brand'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { t, tm, locale } = useI18n()
const { formatRole, roleIcon } = useRoleLabel()

// Form references
const formRef = ref()
const registerFormRef = ref()

// State management
const loading = ref(false)
const oidcLoading = ref(false)
const isRegisterMode = ref(false)
const showLanguageMenu = ref(false)
const oidcEnabled = ref(false)
const oidcProviderName = ref('')
// Wait for configuration before showing a registration entry.
const registrationMode = ref('')
const registrationEnabled = computed(() => canRegister(registrationMode.value, !!inviteLookup.value))
const complexPasswordEnabled = ref(false)

// invite-link state. When the URL carries ?token=xxx we resolve it to
// the originating tenant + role and switch the form into a "register
// via invitation" mode when invitation registration is enabled. Submitting the register
// form with this set hits /auth/register-by-invite (auto-login on
// success) instead of /auth/register.
const inviteToken = ref('')
const inviteLookup = ref<InviteLookup | null>(null)
const inviteLookupError = ref('')
const inviteLookupLoading = ref(false)

// Language options
const languageOptions = [
  { value: 'zh-CN', label: '简体中文', shortLabel: '中文', flag: '🇨🇳' },
  { value: 'zh-TW', label: '繁體中文', shortLabel: '繁中', flag: '繁' },
  { value: 'en-US', label: 'English', shortLabel: 'EN', flag: '🇺🇸' },
  { value: 'ru-RU', label: 'Русский', shortLabel: 'RU', flag: '🇷🇺' },
  { value: 'ko-KR', label: '한국어', shortLabel: '한국어', flag: '🇰🇷' },
  { value: 'ja-JP', label: '日本語', shortLabel: '日本語', flag: '🇯🇵' }
]

const currentLanguage = computed(() => locale.value)
const oidcLoginText = computed(() => {
  if (oidcProviderName.value) {
    return t('auth.oidcLoginWithProvider', { provider: oidcProviderName.value })
  }
  return t('auth.oidcLogin')
})
const currentLangOption = computed(() => languageOptions.find(l => l.value === currentLanguage.value))

// Login form data
const formData = reactive<{ [key: string]: any }>({
  email: '',
  password: '',
})

// Register form data
const registerData = reactive<{ [key: string]: any }>({
  username: '',
  email: '',
  password: '',
  confirmPassword: ''
})

// Login form validation rules
const formRules = computed(() => ({
  email: [
    { required: true, message: t('auth.emailRequired'), type: 'error' },
    { email: true, message: t('auth.emailInvalid'), type: 'error' }
  ],
  password: [
    { required: true, message: t('auth.passwordRequired'), type: 'error' },
    { min: 8, message: t('auth.passwordMinLength'), type: 'error' },
    { max: 32, message: t('auth.passwordMaxLength'), type: 'error' }
  ],
}))

// Register form validation rules
const registerRules = computed(() => ({
  username: [
    { required: true, message: t('auth.usernameRequired'), type: 'error' },
    { min: 2, message: t('auth.usernameMinLength'), type: 'error' },
    { max: 20, message: t('auth.usernameMaxLength'), type: 'error' },
    {
      pattern: /^[a-zA-Z0-9_\u4e00-\u9fa5]+$/,
      message: t('auth.usernameInvalid'),
      type: 'error'
    }
  ],
  email: [
    { required: true, message: t('auth.emailRequired'), type: 'error' },
    { email: true, message: t('auth.emailInvalid'), type: 'error' }
  ],
  password: newPasswordRules(t, complexPasswordEnabled.value),
  confirmPassword: [
    { required: true, message: t('auth.confirmPasswordRequired'), type: 'error' },
    {
      validator: (val: string) => val === registerData.password,
      message: t('auth.passwordMismatch'),
      type: 'error'
    }
  ]
}))

// Toggle login/register mode
const toggleMode = () => {
  isRegisterMode.value = !isRegisterMode.value && registrationEnabled.value

  Object.keys(registerData).forEach(key => {
    (registerData as any)[key] = ''
  })
}

// Toggle language menu
const toggleLanguageMenu = () => {
  showLanguageMenu.value = !showLanguageMenu.value
}

// Select language
const selectLanguage = (lang: string) => {
  locale.value = lang
  localStorage.setItem('locale', lang)
  showLanguageMenu.value = false
  MessagePlugin.success(t('language.languageSaved'))
}

// Close language menu when clicking outside
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (!target.closest('.language-switch')) {
    showLanguageMenu.value = false
  }
}

// Add click outside listener
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})

const persistLoginResponse = async (response: any, skipRedirect = false) => {
  // Backend renamed `tenant` to `active_tenant` and added `memberships`
  // when tenant-level RBAC landed (issue #1303). The two are otherwise
  // identical — `active_tenant` is the tenant whose ID is encoded in the
  // JWT, defaulting to the user's home tenant on a fresh login.
  const activeTenant = response.active_tenant || response.tenant
  if (response.user && response.token) {
    // user.tenant_id must be the user's HOME tenant (the immutable row
    // on the users table); useHomeTenant() and the home-badge logic both
    // assume so. The ACTIVE tenant (which can differ from home when the
    // server honoured a remembered last-active-tenant preference) is
    // expressed separately via setSelectedTenant below.
    const homeTenantIdRaw = response.user.tenant_id ?? activeTenant?.id ?? ''
    authStore.setUser(userInfoFromApi(response.user, homeTenantIdRaw))
    authStore.setToken(response.token)
    if (response.refresh_token) {
      authStore.setRefreshToken(response.refresh_token)
    }
    if (activeTenant) {
      authStore.setTenant({
        id: String(activeTenant.id) || '',
        name: activeTenant.name || '',
        owner_id: response.user.id || '',
        created_at: activeTenant.created_at || new Date().toISOString(),
        updated_at: activeTenant.updated_at || new Date().toISOString()
      })
    } else {
      authStore.setTenant(null)
    }
    if (Array.isArray(response.memberships)) {
      authStore.setMemberships(response.memberships)
    }
    // If the backend dropped us into a non-home tenant (honoured a
    // remembered "last active tenant" preference), set the override so
    // subsequent requests carry X-Tenant-ID and the UI stays consistent.
    // Otherwise clear any stale override left in localStorage by a
    // previous session for a different account.
    const activeIdNum = Number(activeTenant?.id)
    const homeIdNum = Number(homeTenantIdRaw)
    if (Number.isFinite(activeIdNum) && Number.isFinite(homeIdNum) && activeIdNum !== homeIdNum) {
      authStore.setSelectedTenant(activeIdNum, activeTenant?.name || null)
    } else {
      authStore.setSelectedTenant(null, null)
    }
  }

  // Pull runtime capabilities (including whether ordinary users may create
  // workspaces) before entering the main UI so create actions never flash
  // briefly when the deployment is invitation-only.
  await authStore.refreshFromAuthMe()
  await nextTick()
  if (skipRedirect) return
  router.replace(authStore.hasValidTenant ? '/platform/home' : '/onboarding/workspace')
}

const getBackendOIDCRedirectURI = () => `${window.location.origin}${getApiBaseUrl()}/api/v1/auth/oidc/callback`

const loadOIDCConfig = async () => {
  try {
    const response = await getOIDCConfig()
    oidcEnabled.value = !!response.success && !!response.enabled
    oidcProviderName.value = response.provider_display_name || ''
  } catch {
    oidcEnabled.value = false
    oidcProviderName.value = ''
  }
}

// Registration stays hidden if configuration cannot be loaded.
const loadAuthConfig = async () => {
  try {
    const response = await getAuthConfig()
    registrationMode.value = response.success ? response.registration_mode : ''
    complexPasswordEnabled.value = response.complex_password_enabled
  } catch {
    registrationMode.value = ''
    complexPasswordEnabled.value = false
  }
}

const handleOIDCLogin = async () => {
  try {
    oidcLoading.value = true
    const response = await getOIDCAuthorizationURL(getBackendOIDCRedirectURI())
    const authorizationURL = response.authorization_url

    if (!response.success || !authorizationURL) {
      MessagePlugin.error(response.message || t('auth.oidcLoginFailed'))
      return
    }

    // 跳转 IdP 会丢失 URL 中的 token，暂存到 sessionStorage，回调后由 App.vue 兑换。
    if (inviteToken.value) {
      sessionStorage.setItem('weknora_pending_invite_token', inviteToken.value)
    }
    window.location.href = authorizationURL
  } catch (error: any) {
    console.error('OIDC 登录跳转失败:', error)
    MessagePlugin.error(error.message || t('auth.oidcLoginFailed'))
  } finally {
    oidcLoading.value = false
  }
}

// 用 token 加入空间并进入应用。会话此时已有效，故即便 token 失效也照常进入（避免困在登录页）。
const acceptAndEnter = async (token: string) => {
  loading.value = true
  try {
    const result = await authStore.acceptInvitationByTokenAndRefresh(token)
    if (result.ok) {
      MessagePlugin.success(t('inviteRegister.joined'))
    } else {
      MessagePlugin.warning(t('inviteRegister.invalidBody'))
    }
  } catch {
    MessagePlugin.warning(t('inviteRegister.invalidBody'))
  } finally {
    loading.value = false
    await nextTick()
    router.replace('/platform/home')
  }
}

// Handle login
const handleLogin = async () => {
  try {
    const valid = await formRef.value?.validate()
    if (valid !== true) return

    loading.value = true

    const response = await login({
      email: formData.email,
      password: formData.password,
    })

    if (response.success) {
      if (inviteToken.value) {
        // 从邀请链接登录：持久化会话后兑换 token 并进入对应空间。
        await persistLoginResponse(response, true)
        await acceptAndEnter(inviteToken.value)
        return
      }
      await persistLoginResponse(response)
      notifyLoginSuccess(response, t, tm, formatRole, roleIcon)
    } else {
      MessagePlugin.error(response.message || t('auth.loginError'))
    }
  } catch (error: any) {
    console.error('登录错误:', error)
    MessagePlugin.error(error.message || t('auth.loginErrorRetry'))
  } finally {
    loading.value = false
  }
}

// Handle registration. Dispatches based on whether the user arrived
// with a share-link token: with token -> register-by-invite (auto-
// login on success); without -> the normal self-service register
// (drops back to the login form for the user to sign in).
const handleRegister = async () => {
  if (!registrationEnabled.value) return
  try {
    const valid = await registerFormRef.value?.validate()
    if (valid !== true) return

    loading.value = true

    if (inviteToken.value) {
      const response = await registerByInvite({
        token: inviteToken.value,
        username: registerData.username,
        email: registerData.email,
        password: registerData.password,
      })
      if (!response.success) {
        MessagePlugin.error(response.message || t('auth.registerFailed'))
        return
      }
      MessagePlugin.success(t('auth.registerSuccess'))
      // register-by-invite returns the same shape as login (token +
      // active_tenant + memberships), so reuse the login persistence
      // path — same store writes, same redirect target.
      await persistLoginResponse(response)
      return
    }

    const response = await register({
      username: registerData.username,
      email: registerData.email,
      password: registerData.password
    })

    if (response.success) {
      MessagePlugin.success(t('auth.registerSuccess'))

      // Switch to login mode and fill in email
      isRegisterMode.value = false
      formData.email = registerData.email

      // Clear register form
      Object.keys(registerData).forEach(key => {
        (registerData as any)[key] = ''
      })
    } else {
      MessagePlugin.error(response.message || t('auth.registerFailed'))
    }
  } catch (error: any) {
    console.error('注册错误:', error)
    MessagePlugin.error(error.message || t('auth.registerError'))
  } finally {
    loading.value = false
  }
}

// Check if already logged in; for lite edition, attempt transparent auto-setup
onMounted(async () => {
  // Share-link landing: ?token=xxx switches the form into invite-
  // register mode before any other auto-flow (logged-in redirect /
  // auto-setup / OIDC) gets a chance to redirect. Resolution failure
  // surfaces inline; the user can still log in normally if they
  // already have an account. We check this BEFORE the isLoggedIn
  // redirect so an existing session doesn't bounce the user to
  // /platform (and possibly back to /login if the session is stale),
  // dropping the invite token along the way.
  const tokenFromQuery = String(route.query.token || '').trim()
  if (tokenFromQuery) {
    inviteToken.value = tokenFromQuery
    inviteLookupLoading.value = true
    // 1. 先校验 token：无效/过期则停在登录页报错，不进注册模式。
    try {
      const resp = await getInvitationByToken(tokenFromQuery)
      if (resp.success && resp.data) {
        inviteLookup.value = resp.data
      } else {
        inviteLookupError.value = resp.message || t('inviteRegister.invalidBody')
        loadOIDCConfig()
        loadAuthConfig()
        return
      }
    } catch {
      inviteLookupError.value = t('inviteRegister.invalidBody')
      loadOIDCConfig()
      loadAuthConfig()
      return
    } finally {
      inviteLookupLoading.value = false
    }

    // 2. 已登录则直接兑换 token 进入空间（三种模式通用）。
    if (authStore.isLoggedIn && (await authStore.refreshFromAuthMe())) {
      await acceptAndEnter(tokenFromQuery)
      return
    }

    // 3. Only enabled registration modes may create an invited account.
    await loadAuthConfig()
    isRegisterMode.value = registrationEnabled.value
    loadOIDCConfig()
    return
  }

  if (authStore.isLoggedIn) {
    router.replace('/platform/home')
    return
  }

  localStorage.removeItem('weknora_auto_setup_failed')
  try {
    const response = await autoSetup()
    if (response.success) {
      authStore.setLiteMode(true)
      await persistLoginResponse(response)
      return
    }
  } catch {
    // Auto-setup may be unavailable outside the native Lite shell.
  }

  loadOIDCConfig()
  loadAuthConfig()
})
</script>

<style lang="less" scoped>
/* Askademic 登录页 —— editorial-greeting 方向。
   石色地面 + 白纸切片（data-ui）+ 表单列；主按钮近黑；强调色深青。
   遵循 .taste-skill/direction.json 与 app-screens 房规：无渐变、无阴影（静止元素）、无眉标。 */
.login-page {
  min-height: 100svh;
  background: #f5f5f4; /* 低饱和 stone，暖而不奶油 */
  display: flex;
  flex-direction: column;
  position: relative;

  /* 本应用没有全局 border-box；本页自带的盒模型统一收敛，
     避免 width:100% + padding 在窄屏横向溢出（inspect overflow FAIL） */
  *,
  *::before,
  *::after {
    box-sizing: border-box;
  }
}

.lang-switch {
  position: absolute;
  top: 20px;
  right: 24px;
  z-index: 10;

  .header-link {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 34px;
    padding: 0 12px;
    border: 1px solid var(--ad-border);
    border-radius: 999px;
    background: #fff;
    color: var(--ad-text);
    font: 400 13px/1 var(--app-font-family);
    cursor: pointer;
  }
  .header-link:hover { border-color: var(--ad-control); }
}

.language-dropdown {
  position: absolute;
  top: 40px;
  right: 0;
  min-width: 150px;
  background: #fff;
  border: 1px solid var(--ad-border);
  border-radius: 14px;
  padding: 6px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08); /* 浮层允许阴影 */

  .language-option {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    border-radius: 8px;
    font: 400 13px/1.2 var(--app-font-family);
    color: var(--ad-text);
    cursor: pointer;

    &:hover { background: var(--ad-surface); }
    &.active { background: var(--ad-surface); font-weight: 500; }
    .check-icon { margin-left: auto; color: var(--ad-accent); }
    .lang-label { flex: 1; }
  }
}

.stage {
  flex: 1;
  display: flex;
  flex-direction: row-reverse; /* DOM 表单在前 → 桌面表单居右，手机表单在上 */
  align-items: center;
  justify-content: center;
  gap: 64px;
  padding: 56px 32px;
  max-width: 1120px;
  margin: 0 auto;
  width: 100%;
}

/* ---------- 表单列 ---------- */
.formcol {
  width: 380px;
  max-width: 100%;
  flex: none;
}

.brand-block {
  margin-bottom: 24px;

  .brand-logo {
    margin-bottom: 10px;
  }
  /* 字标图片即品牌展示,h1 仅保留文档语义(inspect 标题检查) */
  .sr-only-brand {
    position: absolute;
    width: 1px;
    height: 1px;
    margin: -1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }
  .brand-sub {
    font: 400 13px/1.6 var(--app-font-family);
    color: var(--ad-muted);
    margin: 0;
  }
}

.form-card {
  background: transparent;
}

// 登录/注册两态共用的表单节奏:字段间距统一,提交区与字段区分离
.form-card :deep(.t-form__item) {
  margin-bottom: 18px;
}

.form-card :deep(.t-form__item:last-child) {
  margin-bottom: 0; // 按钮的 margin-top 负责提交区间距,不叠加
}

.form-header {
  margin-bottom: 14px;

  .form-title {
    font: 600 17px/1.3 var(--app-font-family);
    color: var(--ad-text);
    margin: 0 0 4px;
  }
  .form-subtitle {
    font: 400 13px/1.6 var(--app-font-family);
    color: var(--ad-muted);
    margin: 0;
  }
  .form-hint {
    font: 400 13px/1.6 var(--app-font-family);
    color: var(--ad-muted);
    margin: 0;
  }
}

.submit-button {
  margin-top: 18px; // 最后一个字段与主按钮之间留呼吸位(TDesign 默认贴死)
  background-color: var(--ad-primary) !important;
  border-color: var(--ad-primary) !important;
  color: #fff !important;
  border-radius: 999px;
  height: 44px;
  font-weight: 500;

  &:hover { background-color: var(--ad-primary-hover) !important; }
  &:active { background-color: var(--ad-primary-active) !important; }
}

.register-cta {
  margin-top: 14px;

  &__divider {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--ad-faint);
    font: 400 12px/1 var(--app-font-family);
    margin: 0 0 10px;

    &::before,
    &::after {
      content: '';
      flex: 1;
      border-top: 1px solid var(--ad-rule);
    }
  }

  &__button {
    border-radius: 999px;
    height: 42px;
  }
}

.oidc-divider {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--ad-faint);
  font: 400 12px/1 var(--app-font-family);
  margin: 14px 0 10px;

  &::before,
  &::after {
    content: '';
    flex: 1;
    border-top: 1px solid var(--ad-rule);
  }
}

.oidc-button {
  border-radius: 999px;
  height: 42px;
}

.form-footer {
  margin-top: 12px;
  font: 400 13px/1.6 var(--app-font-family);
  color: var(--ad-muted);

  .link-button {
    color: var(--ad-accent);
    font-weight: 500;
    text-decoration: none;
    margin-left: 6px;

    &:hover { text-decoration: underline; }
  }
}

.foot {
  margin-top: 22px;
  font: 400 12px/1.6 var(--app-font-family);
  color: var(--ad-faint);

  b { font-weight: 500; color: var(--ad-muted); }
}

/* 邀请横幅（沿用语义，配色对齐令牌） */
.invite-banner {
  display: flex;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--ad-surface);
  border: 1px solid var(--ad-border);
  margin-bottom: 14px;

  &__icon { color: var(--ad-accent); margin-top: 2px; }
  &__title { font: 500 13px/1.5 var(--app-font-family); color: var(--ad-text); }
  &__hint { font: 400 12px/1.5 var(--app-font-family); color: var(--ad-muted); }

  &--error {
    border-color: var(--td-error-color-3);
    color: var(--td-error-color-7);
    font: 400 13px/1.6 var(--app-font-family);
  }
}

/* ---------- 左列：产品切片 ---------- */
.sheet {
  width: 560px;
  max-width: 100%;
  background: var(--ad-ground);
  border: 1px solid var(--ad-border);
  border-radius: var(--ad-radius-card);
  padding: 26px 28px 24px;
}

.mini-brand {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.statement-block {
  .statement {
    font: 400 30px/1.25 var(--app-font-family-serif);
    letter-spacing: 0.2px;
    color: var(--ad-text);
    margin: 0 0 10px;
  }
  .statement-sub {
    font: 400 13px/1.7 var(--app-font-family);
    color: var(--ad-muted);
    margin: 0;
    max-width: 44ch;
  }
}

.cap-panel {
  margin-top: 20px;
  background: var(--ad-surface);
  border-radius: 16px;
  padding: 12px;

  .cap-row {
    background: var(--ad-ground);
    border: 1px solid var(--ad-border);
    border-radius: 12px;
    min-height: 56px;
    display: flex;
    align-items: center;
    padding: 9px 14px;

    & + .cap-row { margin-top: 6px; }
    .cap-title { font: 500 13px/1.4 var(--app-font-family); color: var(--ad-text); }
    .cap-sub { font: 400 12px/1.5 var(--app-font-family); color: var(--ad-muted); }
  }
}

/* ---------- 手机：表单在上，产品切片在下 ---------- */
@media (max-width: 1000px) {
  .stage {
    flex-direction: column;
    gap: 40px;
    padding: 72px 20px 48px;
  }
  .formcol,
  .sheet {
    width: 420px;
  }
  .statement {
    font-size: 26px;
  }
}

@media (max-width: 480px) {
  .stage { padding: 64px 16px 40px; }
  .formcol,
  .sheet { width: 100%; }
  .lang-switch { top: 14px; right: 14px; }
}
</style>
