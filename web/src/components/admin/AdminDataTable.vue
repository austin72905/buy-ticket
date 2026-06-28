<script setup lang="ts">
defineProps<{
  columns: string[]
  empty?: boolean
  emptyText?: string
}>()
</script>

<template>
  <div class="admin-table-wrap">
    <table class="admin-table">
      <thead>
        <tr>
          <th v-for="column in columns" :key="column">{{ column }}</th>
        </tr>
      </thead>
      <tbody v-if="!empty">
        <slot />
      </tbody>
      <tbody v-else>
        <tr>
          <td :colspan="columns.length" class="admin-empty-cell">
            {{ emptyText || 'No data.' }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
