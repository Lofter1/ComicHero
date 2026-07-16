import { computed, ref } from 'vue'
import {
  getUserStatus,
  loginUser,
  logoutUser,
  registerUser,
  requestPasswordReset,
  resendEmailVerification,
  resetPassword,
  setupUsers,
  verifyEmail,
} from '@/api/client.js'

export function useAuthSession({ route, error, onReady }) {
  const authLoading = ref(true)
  const authSaving = ref(false)
  const userStatus = ref(null)
  const authMode = ref('login')
  const loginRequested = ref(false)
  const setupForm = ref({ mode: 'single', name: '', email: '', password: '' })
  const authForm = ref({
    name: '',
    email: '',
    emailConfirmation: '',
    password: '',
    passwordConfirmation: '',
    inviteToken: '',
  })
  const verificationForm = ref({ token: '', email: '', password: '' })
  const passwordResetForm = ref({
    email: '',
    token: '',
    password: '',
    passwordConfirmation: '',
    requested: false,
    completed: false,
  })

  const setupRequired = computed(() => Boolean(userStatus.value?.setupRequired))
  const userMode = computed(() => userStatus.value?.mode || '')
  const registrationMode = computed(() => userStatus.value?.registrationMode || 'invite_only')
  const publicAccess = computed(() => Boolean(userStatus.value?.publicAccess))
  const currentUser = computed(() => userStatus.value?.user || null)
  const isAdmin = computed(() => Boolean(currentUser.value?.isAdmin))
  const isReadOnlyGuest = computed(
    () => userMode.value === 'multi' && publicAccess.value && !currentUser.value,
  )
  const emailVerificationRequired = computed(() =>
    Boolean(userStatus.value?.emailVerificationRequired),
  )
  const passwordResetMode = computed(() => authMode.value === 'forgot')
  const authRequired = computed(
    () =>
      userMode.value === 'multi' &&
      !currentUser.value &&
      !emailVerificationRequired.value &&
      (!publicAccess.value || loginRequested.value),
  )
  const appReady = computed(
    () =>
      Boolean(userStatus.value) &&
      !authLoading.value &&
      !setupRequired.value &&
      !authRequired.value &&
      !emailVerificationRequired.value &&
      !passwordResetMode.value,
  )

  async function loadUserStatus() {
    authLoading.value = true
    error.value = ''
    try {
      userStatus.value = await getUserStatus()
    } catch (err) {
      error.value = err.message
    } finally {
      authLoading.value = false
    }
  }

  async function submitSetup() {
    await withSaving(async () => {
      const payload = { mode: setupForm.value.mode }
      if (setupForm.value.mode === 'multi') {
        payload.name = setupForm.value.name
        payload.email = setupForm.value.email
        payload.password = setupForm.value.password
      }
      userStatus.value = await setupUsers(payload)
      if (!authRequired.value) await onReady?.({ replace: true, force: true })
    })
  }

  async function submitAuth() {
    await withSaving(async () => {
      validateRegistrationForm()
      const payload = { email: authForm.value.email, password: authForm.value.password }
      if (authMode.value === 'register') {
        Object.assign(payload, {
          name: authForm.value.name,
          emailConfirmation: authForm.value.emailConfirmation,
          passwordConfirmation: authForm.value.passwordConfirmation,
        })
        if (registrationMode.value === 'invite_only')
          payload.inviteToken = authForm.value.inviteToken
      }
      userStatus.value =
        authMode.value === 'register' ? await registerUser(payload) : await loginUser(payload)
      if (userStatus.value?.emailVerificationRequired) {
        verificationForm.value.email =
          userStatus.value.emailVerificationEmail || authForm.value.email
        verificationForm.value.password = authForm.value.password
        return
      }
      loginRequested.value = false
      await onReady?.({ replace: true, force: true })
    })
  }

  function validateRegistrationForm() {
    if (authMode.value !== 'register') return
    if (authForm.value.email !== authForm.value.emailConfirmation) {
      throw new Error('Email confirmation must match email.')
    }
    if (authForm.value.password !== authForm.value.passwordConfirmation) {
      throw new Error('Password confirmation must match password.')
    }
  }

  async function submitEmailVerification() {
    await withSaving(async () => {
      userStatus.value = await verifyEmail({ token: verificationForm.value.token })
      verificationForm.value.token = ''
      loginRequested.value = false
      await onReady?.({ replace: true, force: true })
    })
  }

  async function resendVerificationEmail() {
    await withSaving(async () => {
      userStatus.value = await resendEmailVerification({
        email: verificationForm.value.email || userStatus.value?.emailVerificationEmail,
        password: verificationForm.value.password,
      })
    })
  }

  async function verifyEmailFromRouteToken() {
    const token = routeToken()
    if (!token) return
    verificationForm.value.token = token
    await submitEmailVerification()
  }

  function showForgotPassword() {
    authMode.value = 'forgot'
    passwordResetForm.value = {
      email: authForm.value.email,
      token: '',
      password: '',
      passwordConfirmation: '',
      requested: false,
      completed: false,
    }
    error.value = ''
  }

  function showLogin() {
    authMode.value = 'login'
    loginRequested.value = true
    error.value = ''
  }

  function requestLogin() {
    authMode.value = 'login'
    loginRequested.value = true
  }

  async function submitForgotPassword() {
    await withSaving(async () => {
      userStatus.value = await requestPasswordReset({ email: passwordResetForm.value.email })
      passwordResetForm.value.requested = true
    })
  }

  async function submitPasswordReset() {
    await withSaving(async () => {
      if (passwordResetForm.value.password !== passwordResetForm.value.passwordConfirmation) {
        throw new Error('Password confirmation must match password.')
      }
      userStatus.value = await resetPassword({
        token: passwordResetForm.value.token,
        password: passwordResetForm.value.password,
        passwordConfirmation: passwordResetForm.value.passwordConfirmation,
      })
      passwordResetForm.value.completed = true
      authMode.value = 'login'
      authForm.value.email = passwordResetForm.value.email
      authForm.value.password = ''
    })
  }

  function preparePasswordResetFromRouteToken() {
    const token = routeToken()
    if (!token) return
    authMode.value = 'forgot'
    loginRequested.value = true
    passwordResetForm.value.token = token
    passwordResetForm.value.requested = true
    passwordResetForm.value.completed = false
  }

  async function signOut() {
    await withSaving(async () => {
      await logoutUser()
      userStatus.value = await getUserStatus()
      loginRequested.value = false
    })
  }

  async function withSaving(action) {
    authSaving.value = true
    error.value = ''
    try {
      await action()
    } catch (err) {
      error.value = err.message
    } finally {
      authSaving.value = false
    }
  }

  function routeToken() {
    return typeof route.query.token === 'string' ? route.query.token.trim() : ''
  }

  return {
    authLoading,
    authSaving,
    userStatus,
    authMode,
    setupForm,
    authForm,
    verificationForm,
    passwordResetForm,
    setupRequired,
    userMode,
    registrationMode,
    publicAccess,
    currentUser,
    isAdmin,
    isReadOnlyGuest,
    emailVerificationRequired,
    passwordResetMode,
    authRequired,
    appReady,
    loadUserStatus,
    submitSetup,
    submitAuth,
    submitEmailVerification,
    resendVerificationEmail,
    verifyEmailFromRouteToken,
    showForgotPassword,
    showLogin,
    requestLogin,
    submitForgotPassword,
    submitPasswordReset,
    preparePasswordResetFromRouteToken,
    signOut,
  }
}
