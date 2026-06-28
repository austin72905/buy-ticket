<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'

const router = useRouter()
const backoffice = useAdminBackofficeStore()

const columns = ['ID', 'Name', 'Venue', 'Status', 'Organizer', 'Start', 'Sale Start', 'Sale End', 'Actions']

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

async function load() {
  try {
    await backoffice.loadEvents()
  } catch {}
}

onMounted(load)
</script>

<template>
  <AdminShell>
    <section class="admin-page">
      <header class="admin-page-header">
        <div>
          <p class="eyebrow">Events</p>
          <h1>Event Management</h1>
          <p class="small-muted">View events and section inventory.</p>
        </div>
        <Button label="Refresh" icon="pi pi-refresh" severity="secondary" :loading="backoffice.isPending('events')" @click="load" />
      </header>

      <AdminStateBanner
        :loading="backoffice.isPending('events')"
        :error="backoffice.getError('events')"
        loading-text="Loading events..."
        :retryable="true"
        @retry="load"
      />

      <AdminDataTable :columns="columns" :empty="backoffice.events.length === 0" empty-text="No events found.">
        <tr v-for="event in backoffice.events" :key="event.id">
          <td>{{ event.id }}</td>
          <td>{{ event.name }}</td>
          <td>{{ event.venue }}</td>
          <td><AdminStatusTag :status="event.status" kind="event" /></td>
          <td>{{ event.organizer_id }}</td>
          <td>{{ formatDateTime(event.start_at) }}</td>
          <td>{{ formatDateTime(event.sale_start_at) }}</td>
          <td>{{ formatDateTime(event.sale_end_at) }}</td>
          <td>
            <Button label="Sections" severity="danger" size="small" @click="router.push(`/admin/events/${event.id}`)" />
          </td>
        </tr>
      </AdminDataTable>
    </section>
  </AdminShell>
</template>
