export const OrderStatus = {
  PendingPayment: 1,
  Paid: 2,
  Expired: 3,
  Cancelled: 4,
} as const

const labels: Record<number, string> = {
  [OrderStatus.PendingPayment]: 'Pending Payment',
  [OrderStatus.Paid]: 'Paid',
  [OrderStatus.Expired]: 'Expired',
  [OrderStatus.Cancelled]: 'Cancelled',
}

export function orderStatusLabel(status: number) {
  return labels[status] ?? `Unknown (${status})`
}

export function canPayOrder(status: number) {
  return status === OrderStatus.PendingPayment
}

export function orderStatusSeverity(status: number) {
  if (status === OrderStatus.Paid) return 'success'
  if (status === OrderStatus.Expired || status === OrderStatus.Cancelled) return 'danger'
  return 'warning'
}
