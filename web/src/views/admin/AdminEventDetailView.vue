<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import AdminStatusTag from '../../components/admin/AdminStatusTag.vue'
import { SectionStatus, normalizeSectionStatus } from '../../lib/statusValues'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'
import type { AdminSectionResponse } from '../../types/admin'

const route = useRoute()
const router = useRouter()
const backoffice = useAdminBackofficeStore()

const eventId = computed(() => Number(route.params.eventId))
const showCreate = ref(false)
const editingSection = ref<AdminSectionResponse | null>(null)
const form = ref({
  name: '',
  price: 1,
  totalQuantity: 1,
  purchaseLimit: 1,
  status: 1,
})
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
  'Actions',
]

function formatCurrency(value?: number) {
  return new Intl.NumberFormat('zh-TW', {
    style: 'currency',
    currency: 'TWD',
    maximumFractionDigits: 0,
  }).format(value ?? 0)
}

function resetForm() {
  editingSection.value = null
  form.value = {
    name: '',
    price: 1,
    totalQuantity: 1,
    purchaseLimit: 1,
    status: 1,
  }
}

function sectionStatusToNumber(status: string) {
  const normalized = normalizeSectionStatus(status)
  if (normalized === SectionStatus.Inactive) return 2
  if (normalized === SectionStatus.SoldOut) return 3
  return 1
}

function openCreate() {
  resetForm()
  showCreate.value = true
}

function openEdit(section: AdminSectionResponse) {
  editingSection.value = section
  form.value = {
    name: section.name,
    price: section.price,
    totalQuantity: section.total_quantity,
    purchaseLimit: section.purchase_limit,
    status: sectionStatusToNumber(section.status),
  }
  showCreate.value = true
}

async function load() {
  try {
    await backoffice.loadEventDetail(eventId.value)
  } catch {}
}

async function submitSection() {
  try {
    const payload = {
      name: form.value.name,
      price: Number(form.value.price),
      total_quantity: Number(form.value.totalQuantity),
      purchase_limit: Number(form.value.purchaseLimit),
      status: form.value.status,
    }

    if (editingSection.value) {
      await backoffice.editSection(eventId.value, editingSection.value.id, payload)
    } else {
      await backoffice.saveSection(eventId.value, payload)
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
          <p class="eyebrow">Event Detail</p>
          <h1>{{ backoffice.selectedEvent?.name ?? `Event #${eventId}` }}</h1>
          <p class="small-muted">{{ backoffice.selectedEvent?.venue ?? '-' }}</p>
        </div>
        <div class="admin-actions">
          <Button label="Back" severity="secondary" outlined @click="router.push('/admin/events')" />
          <Button label="New Section" severity="danger" @click="openCreate" />
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
          <td>
            <Button label="Edit" severity="secondary" size="small" outlined @click="openEdit(section)" />
          </td>
        </tr>
      </AdminDataTable>
    </section>

    <div v-if="showCreate" class="admin-modal-backdrop">
      <form class="admin-modal" @submit.prevent="submitSection">
        <header class="admin-modal-header">
          <div>
            <h2>{{ editingSection ? 'Edit Section' : 'New Section' }}</h2>
            <p class="small-muted">Create a section under this event.</p>
          </div>
          <Button label="Close" severity="secondary" outlined type="button" @click="showCreate = false; resetForm()" />
        </header>

        <div class="admin-modal-body">
          <label class="admin-field">
            <span>Name</span>
            <input v-model.trim="form.name" class="plain-input" required />
          </label>
          <label class="admin-field">
            <span>Price</span>
            <input v-model.number="form.price" class="plain-input" type="number" min="1" required />
          </label>
          <label class="admin-field">
            <span>Total Quantity</span>
            <input v-model.number="form.totalQuantity" class="plain-input" type="number" min="1" required />
          </label>
          <label class="admin-field">
            <span>Purchase Limit</span>
            <input v-model.number="form.purchaseLimit" class="plain-input" type="number" min="1" required />
          </label>
          <label class="admin-field">
            <span>Status</span>
            <select v-model.number="form.status" class="plain-input">
              <option :value="1">Active</option>
              <option :value="2">Inactive</option>
              <option :value="3">Sold Out</option>
            </select>
          </label>
          <p v-if="backoffice.getError('saveSection')" class="admin-error">
            {{ backoffice.getError('saveSection') }}
          </p>
        </div>

        <footer class="admin-modal-actions">
          <Button label="Cancel" severity="secondary" outlined type="button" @click="showCreate = false; resetForm()" />
          <Button
            :label="editingSection ? 'Save' : 'Create'"
            severity="danger"
            type="submit"
            :loading="backoffice.isPending('saveSection')"
          />
        </footer>
      </form>
    </div>
  </AdminShell>
</template>
