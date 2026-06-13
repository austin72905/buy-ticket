<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { canPayOrder, orderStatusLabel, orderStatusSeverity } from '../lib/orderStatus'
import { useBookingFlowStore } from '../stores/bookingFlow'
import type { OrderResponse } from '../types/api'

const router = useRouter()
const flow = useBookingFlowStore()

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

async function load() {
  try {
    await flow.loadMyOrders()
  } catch {}
}

function viewOrder(order: OrderResponse) {
  flow.selectOrder(order)
  router.push(`/events/${order.event_id}/order`)
}

function payOrder(order: OrderResponse) {
  if (!canPayOrder(order.status)) return

  flow.selectOrder(order)
  router.push(`/events/${order.event_id}/payment`)
}

onMounted(load)
</script>

<template>
  <AppShell>
    <StateBanner
      :loading="flow.isPending('myOrders')"
      :error="flow.getError('myOrders')"
      loading-text="Loading orders..."
      :retryable="true"
      @retry="load"
    />

    <section class="flow-page">
      <div class="detail-heading">
        <span></span>
        <h1>My Orders</h1>
        <span></span>
      </div>

      <div class="orders-toolbar">
        <div>
          <h2>Order History</h2>
          <p class="small-muted">Orders are loaded from the authenticated user session.</p>
        </div>
        <Button
          label="Refresh"
          severity="secondary"
          icon="pi pi-refresh"
          :loading="flow.isPending('myOrders')"
          @click="load"
        />
      </div>

      <div v-if="flow.myOrders.length === 0 && !flow.isPending('myOrders')" class="empty-panel">
        <p>No orders found.</p>
        <Button label="Browse Events" severity="danger" @click="router.push('/events')" />
      </div>

      <div v-else class="order-history">
        <article v-for="order in flow.myOrders" :key="order.id" class="order-history-card">
          <div class="order-history-main">
            <div>
              <span>Order No</span>
              <strong class="mono">{{ order.order_no }}</strong>
            </div>
            <div>
              <span>Status</span>
              <Tag :value="orderStatusLabel(order.status)" :severity="orderStatusSeverity(order.status)" />
            </div>
            <div>
              <span>Amount</span>
              <strong>{{ formatCurrency(order.total_amount) }}</strong>
            </div>
            <div>
              <span>Quantity</span>
              <strong>{{ order.quantity }}</strong>
            </div>
            <div>
              <span>Expires At</span>
              <strong>{{ formatDateTime(order.expires_at) }}</strong>
            </div>
          </div>

          <div class="order-history-actions">
            <Button label="View" severity="secondary" outlined @click="viewOrder(order)" />
            <Button
              v-if="canPayOrder(order.status)"
              label="Pay"
              severity="danger"
              @click="payOrder(order)"
            />
          </div>
        </article>
      </div>
    </section>
  </AppShell>
</template>
