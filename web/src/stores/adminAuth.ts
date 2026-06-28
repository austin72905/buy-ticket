import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { adminLogin, adminLogout, getAdminMe } from '../api/admin'
import type { AdminUserResponse } from '../types/admin'

function toErrorMessage(error: unknown) {
  if (
    typeof error === 'object' &&
    error !== null &&
    'response' in error &&
    typeof error.response === 'object' &&
    error.response !== null &&
    'data' in error.response &&
    typeof error.response.data === 'object' &&
    error.response.data !== null &&
    'error' in error.response.data &&
    typeof error.response.data.error === 'string'
  ) {
    return error.response.data.error
  }

  if (error instanceof Error) return error.message
  return 'Unknown error'
}

export const useAdminAuthStore = defineStore('adminAuth', () => {
  const currentAdmin = ref<AdminUserResponse | null>(null)
  const loading = ref(false)
  const error = ref('')

  const isSuperAdmin = computed(() => currentAdmin.value?.role === 'SUPER_ADMIN')
  const isEventAdmin = computed(() => currentAdmin.value?.role === 'EVENT_ADMIN')

  async function run<T>(task: () => Promise<T>) {
    loading.value = true
    error.value = ''

    try {
      return await task()
    } catch (nextError) {
      error.value = toErrorMessage(nextError)
      throw nextError
    } finally {
      loading.value = false
    }
  }

  async function loadAdminMe() {
    currentAdmin.value = await run(getAdminMe)
  }

  async function loginAdmin(input: { email: string; password: string }) {
    currentAdmin.value = await run(() => adminLogin(input))
  }

  async function logoutAdmin() {
    await run(adminLogout)
    currentAdmin.value = null
  }

  function clearAdmin() {
    currentAdmin.value = null
  }

  return {
    clearAdmin,
    currentAdmin,
    error,
    isEventAdmin,
    isSuperAdmin,
    loading,
    loadAdminMe,
    loginAdmin,
    logoutAdmin,
  }
})
