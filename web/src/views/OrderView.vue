<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { useBookingFlowStore } from '../stores/bookingFlow'

const route = useRoute()
const router = useRouter()
const flow = useBookingFlowStore()

const eventId = computed(() => Number(route.params.eventId))

function formatDateTime(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatCurrency(value?: number) {
  return new Intl.NumberFormat('zh-TW', {
    style: 'currency',
    currency: 'TWD',
    maximumFractionDigits: 0,
  }).format(value ?? 0)
}

async function refreshOrder() {
  try {
    await flow.fetchOrderAction()
  } catch {}
}
</script>

<template>
  <AppShell>
    <StateBanner
      :loading="flow.isPending('fetchOrder')"
      :error="flow.getError('fetchOrder')"
      loading-text="Refreshing order..."
      :retryable="true"
      @retry="refreshOrder"
    />

    <section class="flow-page">
      <div class="detail-heading">
        <span></span>
        <h1>Order Review</h1>
        <span></span>
      </div>

      <div v-if="!flow.order" class="empty-panel">
        <p>No order found. Create an order from the reservation page.</p>
        <Button label="Back to Reservation" severity="danger" @click="router.push(`/events/${eventId}/reservation`)" />
      </div>

      <div v-else class="flow-layout">
        <section class="summary-card">
          <h2>Pending Order</h2>
          <div class="status-list">
            <div>
              <span>Order No</span>
              <strong class="mono">{{ flow.order.order_no }}</strong>
            </div>
            <div>
              <span>Status</span>
              <Tag :value="String(flow.order.status)" severity="warning" />
            </div>
            <div>
              <span>Quantity</span>
              <strong>{{ flow.order.quantity }}</strong>
            </div>
            <div>
              <span>Total</span>
              <strong>{{ formatCurrency(flow.order.total_amount) }}</strong>
            </div>
            <div>
              <span>Payment Deadline</span>
              <strong>{{ formatDateTime(flow.order.expires_at) }}</strong>
            </div>
          </div>
        </section>

        <aside class="action-panel">
          <h2>Ready For Payment</h2>
          <p>
            Confirm the order amount before moving to payment. The payment page calls the backend payment API.
          </p>
          <Button label="Refresh Order" severity="secondary" @click="refreshOrder" />
          <Button label="Continue To Payment" severity="danger" @click="router.push(`/events/${eventId}/payment`)" />
        </aside>
      </div>
    </section>
  </AppShell>
</template>
