<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'

import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import { useAdminAuthStore } from '../../stores/adminAuth'

const route = useRoute()
const router = useRouter()
const adminAuth = useAdminAuthStore()

const email = ref('')
const password = ref('')

function redirectTarget() {
  const redirect = route.query.redirect
  return typeof redirect === 'string' && redirect.startsWith('/admin') ? redirect : '/admin/orders'
}

async function login() {
  await adminAuth.loginAdmin({
    email: email.value,
    password: password.value,
  })
  await router.push(redirectTarget())
}
</script>

<template>
  <main class="admin-login-page">
    <form class="admin-login-card" @submit.prevent="login">
      <div>
        <p class="eyebrow">Backoffice</p>
        <h1>Admin Login</h1>
        <p class="small-muted">Login with an admin account to manage orders and events.</p>
      </div>

      <AdminStateBanner
        :loading="adminAuth.loading"
        :error="adminAuth.error"
        loading-text="Signing in..."
      />

      <label class="admin-field">
        <span>Email</span>
        <input v-model.trim="email" class="plain-input" type="email" autocomplete="email" required />
      </label>

      <label class="admin-field">
        <span>Password</span>
        <input v-model="password" class="plain-input" type="password" autocomplete="current-password" required />
      </label>

      <Button label="Login" type="submit" severity="danger" :loading="adminAuth.loading" />
    </form>
  </main>
</template>
