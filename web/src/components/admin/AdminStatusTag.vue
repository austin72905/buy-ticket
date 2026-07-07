<script setup lang="ts">
import Tag from 'primevue/tag'
import {
  AdminStatus,
  EventStatus,
  OrderStatus,
  SectionStatus,
  normalizeAdminStatus,
  normalizeEventStatus,
  normalizeOrderStatus,
  normalizeSectionStatus,
} from '../../lib/statusValues'

const props = defineProps<{
  status: number | string
  kind?: 'order' | 'event' | 'section' | 'admin' | 'audit'
}>()

function label() {
  if (props.kind === 'order') {
    const labels: Record<string, string> = {
      [OrderStatus.PendingPayment]: 'Pending Payment',
      [OrderStatus.Paid]: 'Paid',
      [OrderStatus.Expired]: 'Expired',
      [OrderStatus.Cancelled]: 'Cancelled',
    }
    const status = normalizeOrderStatus(props.status)
    return labels[status] ?? status
  }

  if (props.kind === 'event') {
    const labels: Record<string, string> = {
      [EventStatus.Draft]: 'Draft',
      [EventStatus.Published]: 'Published',
      [EventStatus.OnSale]: 'On Sale',
      [EventStatus.Ended]: 'Ended',
    }
    const status = normalizeEventStatus(props.status)
    return labels[status] ?? status
  }

  if (props.kind === 'section') {
    const labels: Record<string, string> = {
      [SectionStatus.Active]: 'Active',
      [SectionStatus.Inactive]: 'Inactive',
      [SectionStatus.SoldOut]: 'Sold Out',
    }
    const status = normalizeSectionStatus(props.status)
    return labels[status] ?? status
  }

  if (props.kind === 'admin') {
    const labels: Record<string, string> = {
      [AdminStatus.Active]: 'Active',
      [AdminStatus.Disabled]: 'Disabled',
    }
    const status = normalizeAdminStatus(props.status)
    return labels[status] ?? status
  }

  return String(props.status)
}

function severity() {
  if (props.kind === 'order') {
    const status = normalizeOrderStatus(props.status)
    if (status === OrderStatus.Paid) return 'success'
    if (status === OrderStatus.Expired || status === OrderStatus.Cancelled) return 'danger'
    return 'warning'
  }

  if (props.kind === 'event') {
    const status = normalizeEventStatus(props.status)
    if (status === EventStatus.OnSale || status === EventStatus.Published) return 'success'
    if (status === EventStatus.Ended) return 'danger'
    return 'secondary'
  }

  if (props.kind === 'section') {
    const status = normalizeSectionStatus(props.status)
    if (status === SectionStatus.Active) return 'success'
    if (status === SectionStatus.SoldOut) return 'danger'
    return 'secondary'
  }

  if (props.kind === 'admin') {
    const status = normalizeAdminStatus(props.status)
    if (status === AdminStatus.Active) return 'success'
    if (status === AdminStatus.Disabled) return 'danger'
    return 'secondary'
  }

  return 'info'
}
</script>

<template>
  <Tag :value="label()" :severity="severity()" />
</template>
