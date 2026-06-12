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

const name = ref('')
const email = ref('')
const password = ref('')

function redirectTarget() {
  const redirect = route.query.redirect
  return typeof redirect === 'string' && redirect.startsWith('/') ? redirect : '/events'
}

async function submit() {
  await flow.registerAction({
    name: name.value,
    email: email.value,
    password: password.value,
  })

  await router.push(redirectTarget())
}
</script>

<template>
  <AppShell>
    <section class="auth-page">
      <form class="auth-card" @submit.prevent="submit">
        <div>
          <p class="eyebrow">Member Register</p>
          <h1>Create account</h1>
          <p class="small-muted">Register before joining the queue and submitting orders.</p>
        </div>

        <StateBanner
          :loading="flow.isPending('auth')"
          :error="flow.getError('auth')"
          loading-text="Creating account..."
        />

        <label>
          <span>Name</span>
          <input v-model.trim="name" class="plain-input" autocomplete="name" required />
        </label>

        <label>
          <span>Email</span>
          <input v-model.trim="email" class="plain-input" type="email" autocomplete="email" required />
        </label>

        <label>
          <span>Password</span>
          <input v-model="password" class="plain-input" type="password" autocomplete="new-password" required />
        </label>

        <div class="auth-actions">
          <Button label="Create Account" type="submit" severity="danger" :loading="flow.isPending('auth')" />
        </div>

        <p class="auth-alt">
          Already have an account?
          <RouterLink :to="{ name: 'login', query: route.query }">Login</RouterLink>
        </p>
      </form>
    </section>
  </AppShell>
</template>
