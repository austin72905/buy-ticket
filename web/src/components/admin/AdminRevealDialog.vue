<script setup lang="ts">
import { ref, watch } from 'vue'
import Button from 'primevue/button'

import type { AdminOrderResponse, AdminOrderSensitiveResponse } from '../../types/admin'

const props = defineProps<{
  visible: boolean
  order: AdminOrderResponse | null
  result?: AdminOrderSensitiveResponse | null
  loading?: boolean
  error?: string
}>()

const emit = defineEmits<{
  close: []
  submit: [reason: string]
}>()

const reason = ref('')

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      reason.value = ''
    }
  },
)

function submit() {
  const trimmed = reason.value.trim()
  if (!trimmed) return
  emit('submit', trimmed)
}
</script>

<template>
  <div v-if="visible" class="admin-modal-backdrop">
    <section class="admin-modal">
      <header class="admin-modal-header">
        <div>
          <h2>Reveal Sensitive Data</h2>
          <p v-if="order" class="small-muted">Order {{ order.order_no }}</p>
        </div>
        <Button icon="pi pi-times" rounded text severity="secondary" aria-label="Close" @click="$emit('close')" />
      </header>

      <div class="admin-modal-body">
        <p class="small-muted">A reason is required. The backend writes an audit log.</p>
        <label class="admin-field">
          <span>Reason</span>
          <textarea v-model="reason" rows="4" class="plain-input" />
        </label>

        <p v-if="error" class="admin-error">{{ error }}</p>

        <div v-if="result" class="admin-sensitive-result">
          <div><span>User ID</span><strong>{{ result.user_id }}</strong></div>
          <div><span>Name</span><strong>{{ result.user_name }}</strong></div>
          <div><span>Email</span><strong>{{ result.user_email }}</strong></div>
        </div>
      </div>

      <footer class="admin-modal-actions">
        <Button label="Cancel" severity="secondary" outlined @click="$emit('close')" />
        <Button label="Reveal" severity="danger" :disabled="!reason.trim()" :loading="loading" @click="submit" />
      </footer>
    </section>
  </div>
</template>
