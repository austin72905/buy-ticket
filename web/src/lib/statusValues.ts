export const EventStatus = {
  Draft: 'DRAFT',
  Published: 'PUBLISHED',
  OnSale: 'ON_SALE',
  Ended: 'ENDED',
} as const

export const SectionStatus = {
  Active: 'ACTIVE',
  Inactive: 'INACTIVE',
  SoldOut: 'SOLD_OUT',
} as const

export const QueueStatus = {
  Waiting: 'WAITING',
  Ready: 'READY',
  Expired: 'EXPIRED',
} as const

export const ReservationStatus = {
  Holding: 'HOLDING',
  Confirmed: 'CONFIRMED',
  Expired: 'EXPIRED',
  Cancelled: 'CANCELLED',
} as const

export const OrderStatus = {
  PendingPayment: 'PENDING_PAYMENT',
  Paid: 'PAID',
  Expired: 'EXPIRED',
  Cancelled: 'CANCELLED',
} as const

export const PaymentStatus = {
  Pending: 'PENDING',
  Paid: 'PAID',
  Failed: 'FAILED',
  Refunded: 'REFUNDED',
} as const

export const PaymentAttemptStatus = {
  Processing: 'PROCESSING',
  Succeeded: 'SUCCEEDED',
  Failed: 'FAILED',
  Timeout: 'TIMEOUT',
  Cancelled: 'CANCELLED',
} as const

export const AdminStatus = {
  Active: 'ACTIVE',
  Disabled: 'DISABLED',
} as const

type StatusValue = string | number

const legacyEventStatus: Record<number, string> = {
  1: EventStatus.Draft,
  2: EventStatus.Published,
  3: EventStatus.OnSale,
  4: EventStatus.Ended,
}

const legacySectionStatus: Record<number, string> = {
  1: SectionStatus.Active,
  2: SectionStatus.Inactive,
  3: SectionStatus.SoldOut,
}

const legacyQueueStatus: Record<number, string> = {
  1: QueueStatus.Waiting,
  2: QueueStatus.Ready,
  3: QueueStatus.Expired,
}

const legacyReservationStatus: Record<number, string> = {
  1: ReservationStatus.Holding,
  2: ReservationStatus.Confirmed,
  3: ReservationStatus.Expired,
  4: ReservationStatus.Cancelled,
}

const legacyOrderStatus: Record<number, string> = {
  1: OrderStatus.PendingPayment,
  2: OrderStatus.Paid,
  3: OrderStatus.Expired,
  4: OrderStatus.Cancelled,
}

const legacyPaymentStatus: Record<number, string> = {
  1: PaymentStatus.Pending,
  2: PaymentStatus.Paid,
  3: PaymentStatus.Failed,
  4: PaymentStatus.Refunded,
}

const legacyPaymentAttemptStatus: Record<number, string> = {
  1: PaymentAttemptStatus.Processing,
  2: PaymentAttemptStatus.Succeeded,
  3: PaymentAttemptStatus.Failed,
  4: PaymentAttemptStatus.Timeout,
  5: PaymentAttemptStatus.Cancelled,
}

const legacyAdminStatus: Record<number, string> = {
  1: AdminStatus.Active,
  2: AdminStatus.Disabled,
}

function normalize(status: StatusValue, legacyMap: Record<number, string>) {
  if (typeof status === 'number') return legacyMap[status] ?? String(status)
  return status
}

export function normalizeEventStatus(status: StatusValue) {
  return normalize(status, legacyEventStatus)
}

export function normalizeSectionStatus(status: StatusValue) {
  return normalize(status, legacySectionStatus)
}

export function normalizeQueueStatus(status: StatusValue) {
  return normalize(status, legacyQueueStatus)
}

export function normalizeReservationStatus(status: StatusValue) {
  return normalize(status, legacyReservationStatus)
}

export function normalizeOrderStatus(status: StatusValue) {
  return normalize(status, legacyOrderStatus)
}

export function normalizePaymentStatus(status: StatusValue) {
  return normalize(status, legacyPaymentStatus)
}

export function normalizePaymentAttemptStatus(status: StatusValue) {
  return normalize(status, legacyPaymentAttemptStatus)
}

export function normalizeAdminStatus(status: StatusValue) {
  return normalize(status, legacyAdminStatus)
}
