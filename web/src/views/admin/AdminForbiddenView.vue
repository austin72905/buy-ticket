<script setup lang="ts">
import Button from 'primevue/button'
import { useRouter } from 'vue-router'

import AdminShell from '../../layouts/AdminShell.vue'

const router = useRouter()
</script>

<template>
  <AdminShell>
    <section class="admin-page">
      <div class="admin-empty-state">
        <h1>Forbidden</h1>
        <p>You do not have permission to access this admin resource.</p>
        <Button label="Back to Orders" severity="danger" @click="router.push('/admin/orders')" />
      </div>
    </section>
  </AdminShell>
</template>
