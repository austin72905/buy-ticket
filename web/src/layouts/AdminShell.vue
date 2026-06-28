<script setup lang="ts">
import { RouterLink, useRouter } from 'vue-router'
import Button from 'primevue/button'

import { useAdminAuthStore } from '../stores/adminAuth'

const router = useRouter()
const adminAuth = useAdminAuthStore()

const links = [
  { label: 'Dashboard', to: '/admin/orders' },
  { label: 'Orders', to: '/admin/orders' },
  { label: 'Events', to: '/admin/events' },
  { label: 'Audit Logs', to: '/admin/audit-logs' },
]

async function logout() {
  try {
    await adminAuth.logoutAdmin()
  } catch {
    adminAuth.clearAdmin()
  } finally {
    router.push('/admin/login')
  }
}
</script>

<template>
  <div class="admin-site">
    <aside class="admin-sidebar">
      <RouterLink to="/admin/orders" class="admin-brand">
        <strong>BT Admin</strong>
        <span>Backoffice</span>
      </RouterLink>

      <nav class="admin-nav">
        <RouterLink v-for="link in links" :key="link.label" :to="link.to">
          {{ link.label }}
        </RouterLink>
      </nav>
    </aside>

    <section class="admin-main">
      <header class="admin-topbar">
        <div>
          <strong>{{ adminAuth.currentAdmin?.name ?? 'Admin' }}</strong>
          <span>{{ adminAuth.currentAdmin?.email ?? '-' }}</span>
        </div>
        <div class="admin-topbar-actions">
          <span class="admin-role">{{ adminAuth.currentAdmin?.role ?? '-' }}</span>
          <Button label="Logout" severity="secondary" size="small" :loading="adminAuth.loading" @click="logout" />
        </div>
      </header>

      <main class="admin-content">
        <slot />
      </main>
    </section>
  </div>
</template>
