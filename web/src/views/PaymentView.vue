<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Tag from 'primevue/tag'

import StateBanner from '../components/StateBanner.vue'
import AppShell from '../layouts/AppShell.vue'
import { canPayOrder, orderStatusLabel, orderStatusSeverity } from '../lib/orderStatus'
import { useBookingFlowStore } from '../stores/bookingFlow'

const route = useRoute()
const router = useRouter()
const flow = useBookingFlowStore()

const eventId = computed(() => Number(route.params.eventId))
const paymentMethod = 'credit_card'
const paymentIdempotencyKey = ref('')

function formatCurrency(value?: number) {
  return new Intl.NumberFormat('zh-TW', {
    style: 'currency',
    currency: 'TWD',
    maximumFractionDigits: 0,
  }).format(value ?? 0)
}

function createUUID() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }

  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (char) => {
    const random = Math.floor(Math.random() * 16)
    const value = char === 'x' ? random : (random & 0x3) | 0x8
    return value.toString(16)
  })
}

function ensurePaymentIdempotencyKey() {
  if (!paymentIdempotencyKey.value) {
    paymentIdempotencyKey.value = createUUID()
  }

  return paymentIdempotencyKey.value
}

async function payOrder() {
  if (!flow.order || !canPayOrder(flow.order.status)) return

  try {
    await flow.payOrderAction({
      method: paymentMethod,
      idempotencyKey: ensurePaymentIdempotencyKey(),
    })
    if (flow.payment) {
      paymentIdempotencyKey.value = ''
    }
  } catch {}
}

watch(
  () => flow.order?.id,
  () => {
    paymentIdempotencyKey.value = ''
  },
)
</script>

<template>
  <AppShell>
    <StateBanner
      :loading="flow.isPending('payOrder')"
      :error="flow.getError('payOrder')"
      loading-text="Submitting payment..."
      :retryable="true"
      @retry="payOrder"
    />

    <section class="flow-page">
      <div class="detail-heading">
        <span></span>
        <h1>Payment</h1>
        <span></span>
      </div>

      <div v-if="!flow.order" class="empty-panel">
        <p>No order found. Create an order before payment.</p>
        <Button label="Back to Order" severity="danger" @click="router.push(`/events/${eventId}/order`)" />
      </div>

      <div v-else class="flow-layout">
        <section class="summary-card">
          <h2>Checkout Summary</h2>
          <div class="status-list">
            <div>
              <span>Order No</span>
              <strong class="mono">{{ flow.order.order_no }}</strong>
            </div>
            <div>
              <span>Amount</span>
              <strong>{{ formatCurrency(flow.order.total_amount) }}</strong>
            </div>
            <div>
              <span>Order Status</span>
              <Tag :value="orderStatusLabel(flow.order.status)" :severity="orderStatusSeverity(flow.order.status)" />
            </div>
            <div>
              <span>Payment Status</span>
              <Tag :value="flow.payment ? String(flow.payment.status) : 'Not paid'" severity="success" />
            </div>
            <div v-if="flow.payment">
              <span>Payment No</span>
              <strong class="mono">{{ flow.payment.payment_no }}</strong>
            </div>
          </div>
        </section>

        <aside class="action-panel">
          <h2>Payment Method</h2>
          <div class="readonly-field">
            <span>Method</span>
            <strong>{{ paymentMethod }}</strong>
          </div>
          <Button
            label="Pay Now"
            severity="danger"
            :disabled="Boolean(flow.payment) || !canPayOrder(flow.order.status)"
            :loading="flow.isPending('payOrder')"
            @click="payOrder"
          />
          <p v-if="!canPayOrder(flow.order.status)" class="small-muted">
            Only pending payment orders can be paid.
          </p>
          <div v-if="flow.payment" class="success-panel">
            <strong>Payment completed</strong>
            <span>The order, reservation, and section sale count are updated by the backend.</span>
          </div>
        </aside>
      </div>
    </section>
  </AppShell>
</template>
