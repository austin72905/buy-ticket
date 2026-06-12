<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'

const route = useRoute()
const router = useRouter()
const flow = useBookingFlowStore()

const email = ref('')
const password = ref('')

function redirectTarget() {
  const redirect = route.query.redirect
  return typeof redirect === 'string' && redirect.startsWith('/') ? redirect : '/events'
}

async function submit() {
  await flow.loginAction({
    email: email.value,
    password: password.value,
  })

  await router.push(redirectTarget())
}

async function useDemoAccount() {
  await flow.loginDemoUser()
  await router.push(redirectTarget())
}
</script>

<template>
  <AppShell>
    <section class="auth-page">
      <form class="auth-card" @submit.prevent="submit">
        <div>
          <p class="eyebrow">Member Login</p>
          <h1>Login to buy tickets</h1>
          <p class="small-muted">Use your account before joining the queue or creating an order.</p>
        </div>

        <StateBanner
          :loading="flow.isPending('auth')"
          :error="flow.getError('auth')"
          loading-text="Signing in..."
        />

        <label>
          <span>Email</span>
          <input v-model.trim="email" class="plain-input" type="email" autocomplete="email" required />
        </label>

        <label>
          <span>Password</span>
          <input v-model="password" class="plain-input" type="password" autocomplete="current-password" required />
        </label>

        <div class="auth-actions">
          <Button label="Login" type="submit" severity="danger" :loading="flow.isPending('auth')" />
          <Button
            label="Use Demo Account"
            type="button"
            severity="secondary"
            outlined
            :loading="flow.isPending('auth')"
            @click="useDemoAccount"
          />
        </div>

        <p class="auth-alt">
          No account yet?
          <RouterLink :to="{ name: 'register', query: route.query }">Create account</RouterLink>
        </p>
      </form>
    </section>
  </AppShell>
</template>
