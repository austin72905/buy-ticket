<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'

const backoffice = useAdminBackofficeStore()
const columns = ['ID', 'Name', 'Status', 'Created', 'Updated']
const showCreate = ref(false)
const name = ref('')

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

async function createNextOrganizer() {
  try {
    await backoffice.saveOrganizer({
      name: name.value,
    })
    name.value = ''
    showCreate.value = false
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
          <Button label="New Organizer" severity="danger" @click="showCreate = true" />
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
        </tr>
      </AdminDataTable>
    </section>

    <div v-if="showCreate" class="admin-modal-backdrop">
      <form class="admin-modal" @submit.prevent="createNextOrganizer">
        <header class="admin-modal-header">
          <div>
            <h2>New Organizer</h2>
            <p class="small-muted">Only SUPER_ADMIN can create organizer records.</p>
          </div>
          <Button label="Close" severity="secondary" outlined type="button" @click="showCreate = false" />
        </header>

        <div class="admin-modal-body">
          <label class="admin-field">
            <span>Name</span>
            <input v-model.trim="name" class="plain-input" required />
          </label>
          <p v-if="backoffice.getError('saveOrganizer')" class="admin-error">
            {{ backoffice.getError('saveOrganizer') }}
          </p>
        </div>

        <footer class="admin-modal-actions">
          <Button label="Cancel" severity="secondary" outlined type="button" @click="showCreate = false" />
          <Button label="Create" severity="danger" type="submit" :loading="backoffice.isPending('saveOrganizer')" />
        </footer>
      </form>
    </div>
  </AdminShell>
</template>
