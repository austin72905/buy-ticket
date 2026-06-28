import { ref } from 'vue'
import { defineStore } from 'pinia'

import {
  getAdminEvent,
  listAdminAuditLogs,
  listAdminEventSections,
  listAdminEvents,
  listAdminOrders,
  revealAdminOrderSensitive,
} from '../api/admin'
import type {
  AdminAuditLogResponse,
  AdminCursor,
  AdminEventResponse,
  AdminOrderFilters,
  AdminOrderResponse,
  AdminOrderSensitiveResponse,
  AdminSectionResponse,
} from '../types/admin'

type AdminTaskKey = 'orders' | 'events' | 'eventDetail' | 'auditLogs' | 'reveal'

const pageLimit = 20

function toErrorMessage(error: unknown) {
  if (
    typeof error === 'object' &&
    error !== null &&
    'response' in error &&
    typeof error.response === 'object' &&
    error.response !== null &&
    'data' in error.response &&
    typeof error.response.data === 'object' &&
    error.response.data !== null &&
    'error' in error.response.data &&
    typeof error.response.data.error === 'string'
  ) {
    return error.response.data.error
  }

  if (error instanceof Error) return error.message
  return 'Unknown error'
}

function parseOptionalNumber(value: string) {
  if (!value.trim()) return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

export const useAdminBackofficeStore = defineStore('adminBackoffice', () => {
  const orders = ref<AdminOrderResponse[]>([])
  const orderNextCursor = ref<AdminCursor | null>(null)
  const orderFilters = ref<AdminOrderFilters>({
    eventId: '',
    userId: '',
    status: '',
  })

  const events = ref<AdminEventResponse[]>([])
  const selectedEvent = ref<AdminEventResponse | null>(null)
  const selectedSections = ref<AdminSectionResponse[]>([])

  const auditLogs = ref<AdminAuditLogResponse[]>([])
  const auditNextCursor = ref<AdminCursor | null>(null)

  const revealedOrder = ref<AdminOrderSensitiveResponse | null>(null)
  const pending = ref<Record<string, boolean>>({})
  const errors = ref<Record<string, string>>({})

  function isPending(key: AdminTaskKey) {
    return Boolean(pending.value[key])
  }

  function getError(key: AdminTaskKey) {
    return errors.value[key] ?? ''
  }

  function clearError(key?: AdminTaskKey) {
    if (!key) {
      errors.value = {}
      return
    }

    const next = { ...errors.value }
    delete next[key]
    errors.value = next
  }

  async function runTask<T>(key: AdminTaskKey, task: () => Promise<T>) {
    pending.value = { ...pending.value, [key]: true }
    clearError(key)

    try {
      return await task()
    } catch (error) {
      errors.value = {
        ...errors.value,
        [key]: toErrorMessage(error),
      }
      throw error
    } finally {
      pending.value = { ...pending.value, [key]: false }
    }
  }

  function orderQuery(cursor?: AdminCursor | null) {
    return {
      limit: pageLimit,
      cursor_created_at: cursor?.created_at,
      cursor_id: cursor?.id,
      event_id: parseOptionalNumber(orderFilters.value.eventId),
      user_id: parseOptionalNumber(orderFilters.value.userId),
      status: parseOptionalNumber(orderFilters.value.status),
    }
  }

  async function loadOrders() {
    const data = await runTask('orders', () => listAdminOrders(orderQuery()))
    orders.value = data.items
    orderNextCursor.value = data.next_cursor ?? null
  }

  async function loadMoreOrders() {
    if (!orderNextCursor.value) return

    const data = await runTask('orders', () => listAdminOrders(orderQuery(orderNextCursor.value)))
    orders.value = [...orders.value, ...data.items]
    orderNextCursor.value = data.next_cursor ?? null
  }

  async function loadEvents() {
    events.value = await runTask('events', listAdminEvents)
  }

  async function loadEventDetail(eventId: number) {
    const [event, sections] = await runTask('eventDetail', () =>
      Promise.all([getAdminEvent(eventId), listAdminEventSections(eventId)]),
    )
    selectedEvent.value = event
    selectedSections.value = sections
  }

  async function loadAuditLogs() {
    const data = await runTask('auditLogs', () => listAdminAuditLogs({ limit: pageLimit }))
    auditLogs.value = data.items
    auditNextCursor.value = data.next_cursor ?? null
  }

  async function loadMoreAuditLogs() {
    if (!auditNextCursor.value) return

    const data = await runTask('auditLogs', () =>
      listAdminAuditLogs({
        limit: pageLimit,
        cursor_created_at: auditNextCursor.value?.created_at,
        cursor_id: auditNextCursor.value?.id,
      }),
    )
    auditLogs.value = [...auditLogs.value, ...data.items]
    auditNextCursor.value = data.next_cursor ?? null
  }

  async function revealOrderSensitive(orderId: number, reason: string) {
    revealedOrder.value = await runTask('reveal', () =>
      revealAdminOrderSensitive(orderId, {
        reason,
      }),
    )
  }

  function clearReveal() {
    revealedOrder.value = null
    clearError('reveal')
  }

  return {
    auditLogs,
    auditNextCursor,
    clearError,
    clearReveal,
    errors,
    events,
    getError,
    isPending,
    loadAuditLogs,
    loadEventDetail,
    loadEvents,
    loadMoreAuditLogs,
    loadMoreOrders,
    loadOrders,
    orderFilters,
    orderNextCursor,
    orders,
    pending,
    revealedOrder,
    revealOrderSensitive,
    selectedEvent,
    selectedSections,
  }
})
