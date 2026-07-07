<script setup lang="ts">
import { onMounted } from 'vue'
import Button from 'primevue/button'

import AdminDataTable from '../../components/admin/AdminDataTable.vue'
import AdminLoadMore from '../../components/admin/AdminLoadMore.vue'
import AdminShell from '../../layouts/AdminShell.vue'
import AdminStateBanner from '../../components/admin/AdminStateBanner.vue'
import { useAdminBackofficeStore } from '../../stores/adminBackoffice'

const backoffice = useAdminBackofficeStore()

const columns = [
  'Action',
  'Admin User',
  'Target Type',
  'Target ID',
  'Reason',
  'IP Address',
  'User Agent',
  'Created',
]

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
    await backoffice.loadAuditLogs()
  } catch {}
}

async function loadMore() {
  try {
    await backoffice.loadMoreAuditLogs()
  } catch {}
}

onMounted(load)
</script>

<template>
  <AdminShell>
    <section class="admin-page">
      <header class="admin-page-header">
        <div>
          <p class="eyebrow">Audit Logs</p>
          <h1>Admin Audit Logs</h1>
          <p class="small-muted">Reveal and admin actions are tracked by the backend.</p>
        </div>
        <Button label="Refresh" icon="pi pi-refresh" severity="secondary" :loading="backoffice.isPending('auditLogs')" @click="load" />
      </header>

      <AdminStateBanner
        :loading="backoffice.isPending('auditLogs')"
        :error="backoffice.getError('auditLogs')"
        loading-text="Loading audit logs..."
        :retryable="true"
        @retry="load"
      />

      <AdminDataTable :columns="columns" :empty="backoffice.auditLogs.length === 0" empty-text="No audit logs found.">
        <tr v-for="log in backoffice.auditLogs" :key="log.id">
          <td>{{ log.action }}</td>
          <td>{{ log.admin_user?.name ?? `#${log.admin_user_id}` }}</td>
          <td>{{ log.target_type }}</td>
          <td>{{ log.target?.name ? `${log.target.name} (#${log.target.id})` : `#${log.target_id}` }}</td>
          <td>{{ log.reason || '-' }}</td>
          <td>{{ log.ip_address || '-' }}</td>
          <td class="admin-user-agent">{{ log.user_agent || '-' }}</td>
          <td>{{ formatDateTime(log.created_at) }}</td>
        </tr>
      </AdminDataTable>

      <AdminLoadMore
        :has-more="Boolean(backoffice.auditNextCursor)"
        :loading="backoffice.isPending('auditLogs')"
        @load-more="loadMore"
      />
    </section>
  </AdminShell>
</template>
