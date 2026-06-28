<script setup lang="ts">
import Tag from 'primevue/tag'

const props = defineProps<{
  status: number | string
  kind?: 'order' | 'event' | 'section' | 'admin' | 'audit'
}>()

function label() {
  if (props.kind === 'order') {
    const labels: Record<number, string> = {
      1: 'Pending Payment',
      2: 'Paid',
      3: 'Expired',
      4: 'Cancelled',
    }
    return labels[Number(props.status)] ?? String(props.status)
  }

  return String(props.status)
}

function severity() {
  const value = Number(props.status)

  if (props.kind === 'order') {
    if (value === 2) return 'success'
    if (value === 3 || value === 4) return 'danger'
    return 'warning'
  }

  if (value === 1) return 'success'
  if (value === 0) return 'secondary'
  return 'info'
}
</script>

<template>
  <Tag :value="label()" :severity="severity()" />
</template>
