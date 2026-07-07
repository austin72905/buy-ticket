<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'

const backoffice = useAdminBackofficeStore()
const columns = ['ID', 'Name', 'Email', 'Role', 'Status', 'Organizer', 'Created']
const showCreate = ref(false)
const form = ref({
  name: '',
  email: '',
  password: '',
  role: 'EVENT_ADMIN',
  organizerId: '',
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

function optionalNumber(value: string) {
  if (!value.trim()) return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function resetForm() {
  form.value = {
    name: '',
    email: '',
    password: '',
    role: 'EVENT_ADMIN',
    organizerId: '',
  }
}

async function load() {
  try {
    await Promise.all([backoffice.loadAdminUsers(), backoffice.loadOrganizers()])
  } catch {}
}

async function createUser() {
  try {
    await backoffice.saveAdminUser({
      name: form.value.name,
      email: form.value.email,
      password: form.value.password,
      role: form.value.role,
      organizer_id: optionalNumber(form.value.organizerId),
    })
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
          <p class="eyebrow">Users</p>
          <h1>Admin Users</h1>
          <p class="small-muted">Manage backoffice accounts. This page uses the Swagger admin-users API.</p>
        </div>
        <div class="admin-actions">
          <Button label="New User" severity="danger" @click="showCreate = true" />
          <Button
            label="Refresh"
            icon="pi pi-refresh"
            severity="secondary"
            :loading="backoffice.isPending('users')"
            @click="load"
          />
        </div>
      </header>

      <AdminStateBanner
        :loading="backoffice.isPending('users') || backoffice.isPending('organizers')"
        :error="backoffice.getError('users') || backoffice.getError('organizers')"
        loading-text="Loading admin users..."
        :retryable="true"
        @retry="load"
      />

      <AdminDataTable :columns="columns" :empty="backoffice.adminUsers.length === 0" empty-text="No admin users found.">
        <tr v-for="user in backoffice.adminUsers" :key="user.id">
          <td>{{ user.id }}</td>
          <td>{{ user.name }}</td>
          <td>{{ user.email }}</td>
          <td>{{ user.role }}</td>
          <td><AdminStatusTag :status="user.status" kind="admin" /></td>
          <td>{{ user.organizer_id ?? '-' }}</td>
          <td>{{ formatDateTime(user.created_at) }}</td>
        </tr>
      </AdminDataTable>
    </section>

    <div v-if="showCreate" class="admin-modal-backdrop">
      <form class="admin-modal" @submit.prevent="createUser">
        <header class="admin-modal-header">
          <div>
            <h2>New Admin User</h2>
            <p class="small-muted">SUPER_ADMIN can create SUPER_ADMIN or EVENT_ADMIN accounts.</p>
          </div>
          <Button label="Close" severity="secondary" outlined type="button" @click="showCreate = false" />
        </header>

        <div class="admin-modal-body">
          <label class="admin-field">
            <span>Name</span>
            <input v-model.trim="form.name" class="plain-input" required />
          </label>
          <label class="admin-field">
            <span>Email</span>
            <input v-model.trim="form.email" class="plain-input" type="email" required />
          </label>
          <label class="admin-field">
            <span>Password</span>
            <input v-model="form.password" class="plain-input" type="password" required />
          </label>
          <label class="admin-field">
            <span>Role</span>
            <select v-model="form.role" class="plain-input">
              <option value="EVENT_ADMIN">EVENT_ADMIN</option>
              <option value="SUPER_ADMIN">SUPER_ADMIN</option>
            </select>
          </label>
          <label class="admin-field">
            <span>Organizer</span>
            <select v-model="form.organizerId" class="plain-input">
              <option value="">None</option>
              <option v-for="organizer in backoffice.organizers" :key="organizer.id" :value="String(organizer.id)">
                #{{ organizer.id }} {{ organizer.name }}
              </option>
            </select>
          </label>
          <p v-if="backoffice.getError('saveUser')" class="admin-error">{{ backoffice.getError('saveUser') }}</p>
        </div>

        <footer class="admin-modal-actions">
          <Button label="Cancel" severity="secondary" outlined type="button" @click="showCreate = false" />
          <Button label="Create" severity="danger" type="submit" :loading="backoffice.isPending('saveUser')" />
        </footer>
      </form>
    </div>
  </AdminShell>
</template>
