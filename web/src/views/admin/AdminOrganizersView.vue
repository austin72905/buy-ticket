<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { AdminStatus, normalizeAdminStatus } from '../../lib/statusValues'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'
import type { OrganizerResponse } from '../../types/admin'

const backoffice = useAdminBackofficeStore()
const columns = ['ID', 'Name', 'Status', 'Created', 'Updated', 'Actions']
const showCreate = ref(false)
const editingOrganizer = ref<OrganizerResponse | null>(null)
const form = ref({
  name: '',
  status: 1,
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

async function load() {
  try {
    await backoffice.loadOrganizers()
  } catch {}
}

function adminStatusToNumber(status: string) {
  return normalizeAdminStatus(status) === AdminStatus.Disabled ? 2 : 1
}

function resetForm() {
  editingOrganizer.value = null
  form.value = {
    name: '',
    status: 1,
  }
}

function openCreate() {
  resetForm()
  showCreate.value = true
}

function openEdit(organizer: OrganizerResponse) {
  editingOrganizer.value = organizer
  form.value = {
    name: organizer.name,
    status: adminStatusToNumber(organizer.status),
  }
  showCreate.value = true
}

async function submitOrganizer() {
  try {
    if (editingOrganizer.value) {
      await backoffice.editOrganizer(editingOrganizer.value.id, {
        expected_version: editingOrganizer.value.version,
        name: form.value.name,
        status: form.value.status,
      })
    } else {
      await backoffice.saveOrganizer({
        name: form.value.name,
      })
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
          <p class="eyebrow">Organizers</p>
          <h1>Organizers</h1>
          <p class="small-muted">Manage organizer records for event ownership.</p>
        </div>
        <div class="admin-actions">
          <Button label="New Organizer" severity="danger" @click="openCreate" />
          <Button
            label="Refresh"
            icon="pi pi-refresh"
            severity="secondary"
            :loading="backoffice.isPending('organizers')"
            @click="load"
          />
        </div>
      </header>

      <AdminStateBanner
        :loading="backoffice.isPending('organizers')"
        :error="backoffice.getError('organizers')"
        loading-text="Loading organizers..."
        :retryable="true"
        @retry="load"
      />

      <AdminDataTable :columns="columns" :empty="backoffice.organizers.length === 0" empty-text="No organizers found.">
        <tr v-for="organizer in backoffice.organizers" :key="organizer.id">
          <td>{{ organizer.id }}</td>
          <td>{{ organizer.name }}</td>
          <td><AdminStatusTag :status="organizer.status" kind="admin" /></td>
          <td>{{ formatDateTime(organizer.created_at) }}</td>
          <td>{{ formatDateTime(organizer.updated_at) }}</td>
          <td>
            <Button label="Edit" severity="secondary" size="small" outlined @click="openEdit(organizer)" />
          </td>
        </tr>
      </AdminDataTable>
    </section>

    <div v-if="showCreate" class="admin-modal-backdrop">
      <form class="admin-modal" @submit.prevent="submitOrganizer">
        <header class="admin-modal-header">
          <div>
            <h2>{{ editingOrganizer ? 'Edit Organizer' : 'New Organizer' }}</h2>
            <p class="small-muted">Only SUPER_ADMIN can create organizer records.</p>
          </div>
          <Button label="Close" severity="secondary" outlined type="button" @click="showCreate = false; resetForm()" />
        </header>

        <div class="admin-modal-body">
          <label class="admin-field">
            <span>Name</span>
            <input v-model.trim="form.name" class="plain-input" required />
          </label>
          <label v-if="editingOrganizer" class="admin-field">
            <span>Status</span>
            <select v-model.number="form.status" class="plain-input">
              <option :value="1">Active</option>
              <option :value="2">Disabled</option>
            </select>
          </label>
          <p v-if="backoffice.getError('saveOrganizer')" class="admin-error">
            {{ backoffice.getError('saveOrganizer') }}
          </p>
        </div>

        <footer class="admin-modal-actions">
          <Button label="Cancel" severity="secondary" outlined type="button" @click="showCreate = false; resetForm()" />
          <Button
            :label="editingOrganizer ? 'Save' : 'Create'"
            severity="danger"
            type="submit"
            :loading="backoffice.isPending('saveOrganizer')"
          />
        </footer>
      </form>
    </div>
  </AdminShell>
</template>
