<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import Button from 'primevue/button'

import { useBookingFlowStore } from '../stores/bookingFlow'

const route = useRoute()
const flow = useBookingFlowStore()

const links = [
  { label: 'Events', to: '/events' },
  { label: 'Tickets', to: '/events' },
  { label: 'My Orders', to: '/orders' },
  { label: 'Highlights', to: '/events' },
  { label: 'FAQ', to: '/events' },
  { label: 'Shop', to: '/events' },
]

const step = computed(() => {
  if (route.name === 'payment') return 5
  if (route.name === 'order-review') return 4
  if (route.name === 'reservation-review') return 3
  if (route.name === 'ticket-select') return 2
  if (route.name === 'event-info') return 1
  return 0
})

async function logout() {
  try {
    await flow.logoutAction()
  } catch {}
}

</script>

<template>
  <div class="site">
    <header class="site-header">
      <RouterLink to="/events" class="brand">
        <span class="brand-mark">BT</span>
        <span>
          <strong>buy-ticket</strong>
          <small>ticket demo</small>
        </span>
      </RouterLink>

      <nav class="top-nav">
        <RouterLink v-for="link in links" :key="link.label" :to="link.to">
          {{ link.label }}
        </RouterLink>
      </nav>

      <div class="header-actions">
        <div v-if="flow.currentUser" class="member-chip">
          <span>{{ flow.currentUser.name }}</span>
          <Button icon="pi pi-sign-out" rounded text severity="secondary" aria-label="Logout" @click="logout" />
        </div>
        <div v-else class="auth-panel">
          <RouterLink class="header-link" to="/login">Login</RouterLink>
          <RouterLink class="header-link header-link--primary" to="/register">Register</RouterLink>
        </div>
        <Button icon="pi pi-shopping-cart" rounded text severity="secondary" aria-label="Cart" />
      </div>
    </header>

    <div
      v-if="
        route.name === 'event-info' ||
        route.name === 'ticket-select' ||
        route.name === 'reservation-review' ||
        route.name === 'order-review' ||
        route.name === 'payment'
      "
      class="checkout-steps"
    >
      <div :class="['step', { active: step === 1 }]">Event / Product</div>
      <div :class="['step', { active: step === 2 }]">Seat / Quantity</div>
      <div class="step">Cart</div>
      <div class="step">Checkout</div>
      <div class="step">Complete</div>
    </div>

    <main class="page-shell">
      <slot />
    </main>
  </div>
</template>
