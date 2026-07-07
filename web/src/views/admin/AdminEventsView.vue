<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { EventStatus, normalizeEventStatus } from '../../lib/statusValues'
import { useAdminAuthStore } from '../../stores/adminAuth'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'
import type { AdminEventResponse } from '../../types/admin'

const router = useRouter()
const backoffice = useAdminBackofficeStore()
const adminAuth = useAdminAuthStore()

const columns = ['ID', 'Name', 'Venue', 'Status', 'Organizer', 'Start', 'Sale Start', 'Sale End', 'Actions']
const showCreate = ref(false)
const editingEvent = ref<AdminEventResponse | null>(null)
const form = ref({
  name: '',
  venue: '',
  organizerId: '',
  status: 1,
  startAt: '',
  endAt: '',
  saleStartAt: '',
  saleEndAt: '',
})

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

function toRFC3339(value: string) {
  return new Date(value).toISOString()
}

function optionalNumber(value: string) {
  if (!value.trim()) return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function toDateTimeInput(value: string) {
  if (!value) return ''
  const date = new Date(value)
  const offsetMs = date.getTimezoneOffset() * 60 * 1000
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16)
}

function eventStatusToNumber(status: string) {
  const normalized = normalizeEventStatus(status)
  if (normalized === EventStatus.Published) return 2
  if (normalized === EventStatus.OnSale) return 3
  if (normalized === EventStatus.Ended) return 4
  return 1
}

function resetForm() {
  editingEvent.value = null
  form.value = {
    name: '',
    venue: '',
    organizerId: '',
    status: 1,
    startAt: '',
    endAt: '',
    saleStartAt: '',
    saleEndAt: '',
  }
}

function openCreate() {
  resetForm()
  showCreate.value = true
}

function openEdit(event: AdminEventResponse) {
  editingEvent.value = event
  form.value = {
    name: event.name,
    venue: event.venue,
    organizerId: String(event.organizer_id),
    status: eventStatusToNumber(event.status),
    startAt: toDateTimeInput(event.start_at),
    endAt: toDateTimeInput(event.end_at),
    saleStartAt: toDateTimeInput(event.sale_start_at),
    saleEndAt: toDateTimeInput(event.sale_end_at),
  }
  showCreate.value = true
}

async function load() {
  try {
    await Promise.all([
      backoffice.loadEvents(),
      adminAuth.isSuperAdmin ? backoffice.loadOrganizers() : Promise.resolve(),
    ])
  } catch {}
}

async function submitEvent() {
  try {
    const payload = {
      name: form.value.name,
      venue: form.value.venue,
      organizer_id: optionalNumber(form.value.organizerId),
      status: form.value.status,
      start_at: toRFC3339(form.value.startAt),
      end_at: toRFC3339(form.value.endAt),
      sale_start_at: toRFC3339(form.value.saleStartAt),
      sale_end_at: toRFC3339(form.value.saleEndAt),
    }

    if (editingEvent.value) {
      await backoffice.editEvent(editingEvent.value.id, payload)
    } else {
      await backoffice.saveEvent(payload)
    }

    showCreate.value = false
    resetForm()
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
          <p class="small-muted">View and create events from the admin-events API.</p>
        </div>
        <div class="admin-actions">
          <Button label="New Event" severity="danger" @click="openCreate" />
          <Button
            label="Refresh"
            icon="pi pi-refresh"
            severity="secondary"
            :loading="backoffice.isPending('events')"
            @click="load"
          />
        </div>
      </header>

      <AdminStateBanner
        :loading="backoffice.isPending('events') || backoffice.isPending('organizers')"
        :error="backoffice.getError('events') || backoffice.getError('organizers')"
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
            <div class="admin-actions">
              <Button label="Edit" severity="secondary" size="small" outlined @click="openEdit(event)" />
              <Button label="Sections" severity="danger" size="small" @click="router.push(`/admin/events/${event.id}`)" />
            </div>
          </td>
        </tr>
      </AdminDataTable>
    </section>

    <div v-if="showCreate" class="admin-modal-backdrop">
      <form class="admin-modal" @submit.prevent="submitEvent">
        <header class="admin-modal-header">
          <div>
            <h2>{{ editingEvent ? 'Edit Event' : 'New Event' }}</h2>
            <p class="small-muted">Datetime fields are submitted as RFC3339 timestamps.</p>
          </div>
          <Button label="Close" severity="secondary" outlined type="button" @click="showCreate = false; resetForm()" />
        </header>

        <div class="admin-modal-body">
          <label class="admin-field">
            <span>Name</span>
            <input v-model.trim="form.name" class="plain-input" required />
          </label>
          <label class="admin-field">
            <span>Venue</span>
            <input v-model.trim="form.venue" class="plain-input" required />
          </label>
          <label v-if="adminAuth.isSuperAdmin" class="admin-field">
            <span>Organizer</span>
            <select v-model="form.organizerId" class="plain-input">
              <option value="">Required for SUPER_ADMIN</option>
              <option v-for="organizer in backoffice.organizers" :key="organizer.id" :value="String(organizer.id)">
                #{{ organizer.id }} {{ organizer.name }}
              </option>
            </select>
          </label>
          <label class="admin-field">
            <span>Status</span>
            <select v-model.number="form.status" class="plain-input">
              <option :value="1">Draft</option>
              <option :value="2">Published</option>
              <option :value="3">On Sale</option>
              <option :value="4">Ended</option>
            </select>
          </label>
          <label class="admin-field">
            <span>Start At</span>
            <input v-model="form.startAt" class="plain-input" type="datetime-local" required />
          </label>
          <label class="admin-field">
            <span>End At</span>
            <input v-model="form.endAt" class="plain-input" type="datetime-local" required />
          </label>
          <label class="admin-field">
            <span>Sale Start</span>
            <input v-model="form.saleStartAt" class="plain-input" type="datetime-local" required />
          </label>
          <label class="admin-field">
            <span>Sale End</span>
            <input v-model="form.saleEndAt" class="plain-input" type="datetime-local" required />
          </label>
          <p v-if="backoffice.getError('saveEvent')" class="admin-error">{{ backoffice.getError('saveEvent') }}</p>
        </div>

        <footer class="admin-modal-actions">
          <Button label="Cancel" severity="secondary" outlined type="button" @click="showCreate = false; resetForm()" />
          <Button
            :label="editingEvent ? 'Save' : 'Create'"
            severity="danger"
            type="submit"
            :loading="backoffice.isPending('saveEvent')"
          />
        </footer>
      </form>
    </div>
  </AdminShell>
</template>
