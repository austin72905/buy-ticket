<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import Button from 'primevue/button'
import Card from 'primevue/card'
import Tag from 'primevue/tag'

import { env } from '../config/env'

const route = useRoute()

const links = [
  { label: 'Events', to: '/events' },
  { label: 'Queue', to: '/queue' },
  { label: 'Checkout', to: '/checkout' },
]

const title = computed(() => {
  if (route.name === 'queue') return 'Queue API Playground'
  if (route.name === 'checkout') return 'Checkout API Playground'
  return 'Event API Playground'
})
</script>

<template>
  <div class="shell">
    <header class="hero-panel">
      <div>
        <p class="eyebrow">buy-ticket frontend</p>
        <h1>{{ title }}</h1>
        <p class="hero-copy">
          Vue frontend connected to the current Go API, with shared booking flow state,
          unified request status, and environment-based API config.
        </p>
      </div>
      <div class="hero-meta">
        <Tag severity="info" value="Vue 3 + PrimeVue + Axios" />
        <Tag severity="secondary" :value="`API ${env.apiBaseUrl}`" />
      </div>
    </header>

    <nav class="nav-tabs">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to" class="nav-link">
        <Button :label="link.label" :severity="route.path === link.to ? 'contrast' : 'secondary'" />
      </RouterLink>
    </nav>

    <Card class="content-card">
      <template #content>
        <slot />
      </template>
    </Card>
  </div>
</template>
