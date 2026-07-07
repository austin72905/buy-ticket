import { OrderStatus, normalizeOrderStatus } from './statusValues'

export { OrderStatus }

const labels: Record<string, string> = {
  [OrderStatus.PendingPayment]: 'Pending Payment',
  [OrderStatus.Paid]: 'Paid',
  [OrderStatus.Expired]: 'Expired',
  [OrderStatus.Cancelled]: 'Cancelled',
}

export function orderStatusLabel(status: string | number) {
  const normalized = normalizeOrderStatus(status)
  return labels[normalized] ?? `Unknown (${normalized})`
}

export function canPayOrder(status: string | number) {
  return normalizeOrderStatus(status) === OrderStatus.PendingPayment
}

export function orderStatusSeverity(status: string | number) {
  const normalized = normalizeOrderStatus(status)
  if (normalized === OrderStatus.Paid) return 'success'
  if (normalized === OrderStatus.Expired || normalized === OrderStatus.Cancelled) return 'danger'
  return 'warning'
}
