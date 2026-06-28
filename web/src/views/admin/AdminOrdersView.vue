<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminLoadMore from '../../components/admin/AdminLoadMore.vue'
import AdminRevealDialog from '../../components/admin/AdminRevealDialog.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { useAdminAuthStore } from '../../stores/adminAuth'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'
import type { AdminOrderResponse } from '../../types/admin'

const adminAuth = useAdminAuthStore()
const backoffice = useAdminBackofficeStore()

const revealTarget = ref<AdminOrderResponse | null>(null)

const columns = [
  'Order No',
  'User',
  'Event',
  'Section',
  'Qty',
  'Amount',
  'Status',
  'Created',
  'Expires',
  'Paid',
  'Actions',
]

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
    await backoffice.loadOrders()
  } catch {}
}

async function loadMore() {
  try {
    await backoffice.loadMoreOrders()
  } catch {}
}

function applyFilters() {
  load()
}

function openReveal(order: AdminOrderResponse) {
  backoffice.clearReveal()
  revealTarget.value = order
}

function closeReveal() {
  revealTarget.value = null
  backoffice.clearReveal()
}

async function reveal(reason: string) {
  if (!revealTarget.value) return

  try {
    await backoffice.revealOrderSensitive(revealTarget.value.id, reason)
  } catch {}
}

onMounted(load)
</script>

<template>
  <AdminShell>
    <section class="admin-page">
      <header class="admin-page-header">
        <div>
          <p class="eyebrow">Orders</p>
          <h1>Order Management</h1>
          <p class="small-muted">Keyset pagination with masked user data by default.</p>
        </div>
        <Button label="Refresh" icon="pi pi-refresh" severity="secondary" :loading="backoffice.isPending('orders')" @click="load" />
      </header>

      <AdminStateBanner
        :loading="backoffice.isPending('orders')"
        :error="backoffice.getError('orders')"
        loading-text="Loading admin orders..."
        :retryable="true"
        @retry="load"
      />

      <form class="admin-filters" @submit.prevent="applyFilters">
        <label class="admin-field">
          <span>Event ID</span>
          <input v-model.trim="backoffice.orderFilters.eventId" class="plain-input" inputmode="numeric" />
        </label>
        <label class="admin-field">
          <span>User ID</span>
          <input v-model.trim="backoffice.orderFilters.userId" class="plain-input" inputmode="numeric" />
        </label>
        <label class="admin-field">
          <span>Status</span>
          <select v-model="backoffice.orderFilters.status" class="plain-input">
            <option value="">All</option>
            <option value="1">Pending Payment</option>
            <option value="2">Paid</option>
            <option value="3">Expired</option>
            <option value="4">Cancelled</option>
          </select>
        </label>
        <Button label="Apply" type="submit" severity="danger" />
      </form>

      <AdminDataTable :columns="columns" :empty="backoffice.orders.length === 0" empty-text="No orders found.">
        <tr v-for="order in backoffice.orders" :key="order.id">
          <td class="mono">{{ order.order_no }}</td>
          <td>
            <strong>{{ order.user_name }}</strong>
            <span class="admin-cell-sub">#{{ order.user_id }} · {{ order.user_email }}</span>
          </td>
          <td>{{ order.event_name }}</td>
          <td>{{ order.section_name }}</td>
          <td>{{ order.quantity }}</td>
          <td>{{ formatCurrency(order.total_amount) }}</td>
          <td><AdminStatusTag :status="order.status" kind="order" /></td>
          <td>{{ formatDateTime(order.created_at) }}</td>
          <td>{{ formatDateTime(order.expires_at) }}</td>
          <td>{{ formatDateTime(order.paid_at) }}</td>
          <td>
            <Button
              v-if="adminAuth.isSuperAdmin"
              label="Reveal"
              severity="danger"
              size="small"
              outlined
              @click="openReveal(order)"
            />
          </td>
        </tr>
      </AdminDataTable>

      <AdminLoadMore
        :has-more="Boolean(backoffice.orderNextCursor)"
        :loading="backoffice.isPending('orders')"
        @load-more="loadMore"
      />
    </section>

    <AdminRevealDialog
      :visible="Boolean(revealTarget)"
      :order="revealTarget"
      :result="backoffice.revealedOrder"
      :loading="backoffice.isPending('reveal')"
      :error="backoffice.getError('reveal')"
      @close="closeReveal"
      @submit="reveal"
    />
  </AdminShell>
</template>
