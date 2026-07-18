<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { AdminStatus, normalizeAdminStatus } from '../../lib/statusValues'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'
import type { AdminUserResponse } from '../../types/admin'

const backoffice = useAdminBackofficeStore()
const columns = ['ID', 'Name', 'Email', 'Role', 'Status', 'Organizer', 'Created', 'Actions']
const showCreate = ref(false)
const editingUser = ref<AdminUserResponse | null>(null)
const form = ref({
  name: '',
  email: '',
  password: '',
  role: 'EVENT_ADMIN',
  organizerId: '',
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

function optionalNumber(value: string) {
  if (!value.trim()) return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function resetForm() {
  editingUser.value = null
  form.value = {
    name: '',
    email: '',
    password: '',
    role: 'EVENT_ADMIN',
    organizerId: '',
    status: 1,
  }
}

function adminStatusToNumber(status: string) {
  return normalizeAdminStatus(status) === AdminStatus.Disabled ? 2 : 1
}

function openCreate() {
  resetForm()
  showCreate.value = true
}

function openEdit(user: AdminUserResponse) {
  editingUser.value = user
  form.value = {
    name: user.name,
    email: user.email,
    password: '',
    role: user.role,
    organizerId: user.organizer_id ? String(user.organizer_id) : '',
    status: adminStatusToNumber(user.status),
  }
  showCreate.value = true
}

async function load() {
  try {
    await Promise.all([backoffice.loadAdminUsers(), backoffice.loadOrganizers()])
  } catch {}
}

async function submitUser() {
  try {
    const payload = {
      name: form.value.name,
      email: form.value.email,
      role: form.value.role,
      organizer_id: optionalNumber(form.value.organizerId),
      status: form.value.status,
    }

    if (editingUser.value) {
      await backoffice.editAdminUser(editingUser.value.id, {
        ...payload,
        expected_version: editingUser.value.version,
        password: form.value.password.trim() ? form.value.password : undefined,
      })
    } else {
      await backoffice.saveAdminUser({
        ...payload,
        password: form.value.password,
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
          <p class="eyebrow">Users</p>
          <h1>Admin Users</h1>
          <p class="small-muted">Manage backoffice accounts. This page uses the Swagger admin-users API.</p>
        </div>
        <div class="admin-actions">
          <Button label="New User" severity="danger" @click="openCreate" />
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
          <td>
            <Button label="Edit" severity="secondary" size="small" outlined @click="openEdit(user)" />
          </td>
        </tr>
      </AdminDataTable>
    </section>

    <div v-if="showCreate" class="admin-modal-backdrop">
      <form class="admin-modal" @submit.prevent="submitUser">
        <header class="admin-modal-header">
          <div>
            <h2>{{ editingUser ? 'Edit Admin User' : 'New Admin User' }}</h2>
            <p class="small-muted">Leave password blank when editing to keep the current password.</p>
          </div>
          <Button label="Close" severity="secondary" outlined type="button" @click="showCreate = false; resetForm()" />
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
            <input v-model="form.password" class="plain-input" type="password" :required="!editingUser" />
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
          <label v-if="editingUser" class="admin-field">
            <span>Status</span>
            <select v-model.number="form.status" class="plain-input">
              <option :value="1">Active</option>
              <option :value="2">Disabled</option>
            </select>
          </label>
          <p v-if="backoffice.getError('saveUser')" class="admin-error">{{ backoffice.getError('saveUser') }}</p>
        </div>

        <footer class="admin-modal-actions">
          <Button label="Cancel" severity="secondary" outlined type="button" @click="showCreate = false; resetForm()" />
          <Button
            :label="editingUser ? 'Save' : 'Create'"
            severity="danger"
            type="submit"
            :loading="backoffice.isPending('saveUser')"
          />
        </footer>
      </form>
    </div>
  </AdminShell>
</template>
