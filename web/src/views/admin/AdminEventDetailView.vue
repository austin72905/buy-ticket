<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'

const route = useRoute()
const router = useRouter()
const backoffice = useAdminBackofficeStore()

const eventId = computed(() => Number(route.params.eventId))
const columns = [
  'ID',
  'Section',
  'Price',
  'Total',
  'Reserved',
  'Sold',
  'Available',
  'Limit',
  'Status',
]

function formatCurrency(value?: number) {
  return new Intl.NumberFormat('zh-TW', {
    style: 'currency',
    currency: 'TWD',
    maximumFractionDigits: 0,
  }).format(value ?? 0)
}

async function load() {
  try {
    await backoffice.loadEventDetail(eventId.value)
  } catch {}
}

onMounted(load)
</script>

<template>
  <AdminShell>
    <section class="admin-page">
      <header class="admin-page-header">
        <div>
          <p class="eyebrow">Event Detail</p>
          <h1>{{ backoffice.selectedEvent?.name ?? `Event #${eventId}` }}</h1>
          <p class="small-muted">{{ backoffice.selectedEvent?.venue ?? '-' }}</p>
        </div>
        <div class="admin-actions">
          <Button label="Back" severity="secondary" outlined @click="router.push('/admin/events')" />
          <Button label="Refresh" severity="secondary" icon="pi pi-refresh" :loading="backoffice.isPending('eventDetail')" @click="load" />
        </div>
      </header>

      <AdminStateBanner
        :loading="backoffice.isPending('eventDetail')"
        :error="backoffice.getError('eventDetail')"
        loading-text="Loading event sections..."
        :retryable="true"
        @retry="load"
      />

      <AdminDataTable
        :columns="columns"
        :empty="backoffice.selectedSections.length === 0"
        empty-text="No sections found."
      >
        <tr v-for="section in backoffice.selectedSections" :key="section.id">
          <td>{{ section.id }}</td>
          <td>{{ section.name }}</td>
          <td>{{ formatCurrency(section.price) }}</td>
          <td>{{ section.total_quantity }}</td>
          <td>{{ section.reserved_quantity }}</td>
          <td>{{ section.sold_quantity }}</td>
          <td>{{ section.available_quantity }}</td>
          <td>{{ section.purchase_limit }}</td>
          <td><AdminStatusTag :status="section.status" kind="section" /></td>
        </tr>
      </AdminDataTable>
    </section>
  </AdminShell>
</template>
