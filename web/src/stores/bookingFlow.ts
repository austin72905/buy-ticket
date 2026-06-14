import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import {
  cancelReservation,
  createOrder,
  getAvailability,
  getEvent,
  getMe,
  getOrder,
  getQueueStatus,
  getSaleStatus,
  getSections,
  joinQueue,
  listEvents,
  listMyOrders,
  login,
  logout,
  payOrder,
  register,
  reserveTicket,
} from '../api/booking'
import type {
  EventResponse,
  OrderResponse,
  PaymentResponse,
  QueueStatusResponse,
  ReservationResponse,
  SaleStatusResponse,
  SectionAvailabilityResponse,
  SectionResponse,
  UserResponse,
} from '../types/api'

type TaskKey =
  | 'auth'
  | 'events'
  | 'eventBundle'
  | 'sections'
  | 'queueJoin'
  | 'queueStatus'
  | 'reserve'
  | 'cancelReservation'
  | 'createOrder'
  | 'fetchOrder'
  | 'myOrders'
  | 'payOrder'

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

  if (error instanceof Error) {
    return error.message
  }

  return 'Unknown error'
}

export const useBookingFlowStore = defineStore('bookingFlow', () => {
  const currentUser = ref<UserResponse | null>(null)
  const selectedEventId = ref<number | null>(null)
  const selectedSectionId = ref<number | null>(null)

  const events = ref<EventResponse[]>([])
  const currentEvent = ref<EventResponse | null>(null)
  const sections = ref<SectionResponse[]>([])
  const availability = ref<SectionAvailabilityResponse[]>([])
  const saleStatus = ref<SaleStatusResponse | null>(null)
  const queueStatus = ref<QueueStatusResponse | null>(null)
  const reservation = ref<ReservationResponse | null>(null)
  const order = ref<OrderResponse | null>(null)
  const myOrders = ref<OrderResponse[]>([])
  const payment = ref<PaymentResponse | null>(null)

  const pending = ref<Record<string, boolean>>({})
  const errors = ref<Record<string, string>>({})

  const purchaseToken = computed(() => queueStatus.value?.purchase_token ?? '')
  const queueToken = computed(() => queueStatus.value?.queue_token ?? '')
  const selectedSection = computed(
    () => sections.value.find((section) => section.id === selectedSectionId.value) ?? null,
  )
  const userId = computed(() => currentUser.value?.id ?? 0)

  function isPending(key: TaskKey) {
    return Boolean(pending.value[key])
  }

  function getError(key: TaskKey) {
    return errors.value[key] ?? ''
  }

  function clearError(key?: TaskKey) {
    if (!key) {
      errors.value = {}
      return
    }

    const next = { ...errors.value }
    delete next[key]
    errors.value = next
  }

  function setError(key: TaskKey, message: string) {
    errors.value = {
      ...errors.value,
      [key]: message,
    }
  }

  async function runTask<T>(key: TaskKey, task: () => Promise<T>) {
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

  function resetQueueFlow() {
    queueStatus.value = null
  }

  function resetCheckout() {
    reservation.value = null
    order.value = null
    payment.value = null
  }

  function resetForEventChange() {
    selectedSectionId.value = null
    resetQueueFlow()
    resetCheckout()
  }

  async function loadEvents() {
    const data = await runTask('events', listEvents)
    events.value = data

    if (!selectedEventId.value && data.length > 0) {
      selectedEventId.value = data[0].id
    }
  }

  async function loadMe() {
    currentUser.value = await runTask('auth', getMe)
  }

  async function loginAction(input: { email: string; password: string }) {
    currentUser.value = await runTask('auth', () => login(input))
  }

  async function registerAction(input: { name: string; email: string; password: string }) {
    currentUser.value = await runTask('auth', () => register(input))
  }

  async function logoutAction() {
    await runTask('auth', logout)
    currentUser.value = null
    myOrders.value = []
    resetQueueFlow()
    resetCheckout()
  }

  async function loginDemoUser() {
    const demoUser = {
      name: 'Demo Buyer',
      email: 'demo@buy-ticket.local',
      password: 'password123',
    }

    try {
      await loginAction({
        email: demoUser.email,
        password: demoUser.password,
      })
    } catch {
      await registerAction(demoUser)
    }
  }

  async function loadEventBundle(eventId: number) {
    const [event, nextSections, nextAvailability, nextSaleStatus] = await runTask(
      'eventBundle',
      () =>
        Promise.all([
          getEvent(eventId),
          getSections(eventId),
          getAvailability(eventId),
          getSaleStatus(eventId),
        ]),
    )

    currentEvent.value = event
    sections.value = nextSections
    availability.value = nextAvailability
    saleStatus.value = nextSaleStatus

    if (!selectedSectionId.value || !nextSections.some((section) => section.id === selectedSectionId.value)) {
      selectedSectionId.value = nextSections[0]?.id ?? null
    }
  }

  async function selectEvent(eventId: number) {
    if (selectedEventId.value !== eventId) {
      selectedEventId.value = eventId
      resetForEventChange()
    }

    await loadEventBundle(eventId)
  }

  async function refreshSections() {
    if (!selectedEventId.value) return

    const nextSections = await runTask('sections', () => getSections(selectedEventId.value as number))
    sections.value = nextSections

    if (!selectedSectionId.value || !nextSections.some((section) => section.id === selectedSectionId.value)) {
      selectedSectionId.value = nextSections[0]?.id ?? null
    }
  }

  async function joinQueueAction(payload: {
    clientId: string
    requestId: string
    channel: string
    accessCode?: string
  }) {
    if (!selectedEventId.value) {
      setError('queueJoin', 'Please select an event first.')
      return
    }

    queueStatus.value = await runTask('queueJoin', () =>
      joinQueue({
        event_id: selectedEventId.value as number,
        client_id: payload.clientId,
        request_id: payload.requestId,
        channel: payload.channel,
        access_code: payload.accessCode,
      }),
    )
  }

  async function refreshQueueStatus() {
    if (!queueToken.value) {
      setError('queueStatus', 'Queue token is empty.')
      return
    }

    queueStatus.value = await runTask('queueStatus', () => getQueueStatus(queueToken.value))
  }

  async function reserveTicketAction(input: { quantity: number; holdMinutes: number }) {
    if (!selectedEventId.value || !selectedSectionId.value) {
      setError('reserve', 'Event and section are required.')
      return
    }

    if (!purchaseToken.value) {
      setError('reserve', 'Purchase token is required.')
      return
    }

    const holdUntil = new Date(Date.now() + input.holdMinutes * 60 * 1000).toISOString()

    reservation.value = await runTask('reserve', () =>
      reserveTicket({
        event_id: selectedEventId.value as number,
        section_id: selectedSectionId.value as number,
        quantity: input.quantity,
        hold_until: holdUntil,
        purchase_token: purchaseToken.value,
      }),
    )
    order.value = null
    payment.value = null
  }

  async function createOrderAction(input: { orderNo: string; holdMinutes: number }) {
    if (!reservation.value) {
      setError('createOrder', 'Reservation is required.')
      return
    }
    if (!purchaseToken.value) {
      setError('createOrder', 'Purchase token is required.')
      return
    }

    const expiresAt = new Date(Date.now() + input.holdMinutes * 60 * 1000).toISOString()

    order.value = await runTask('createOrder', () =>
      createOrder({
        reservation_id: reservation.value!.id,
        order_no: input.orderNo,
        expires_at: expiresAt,
        purchase_token: purchaseToken.value,
      }),
    )
    resetQueueFlow()
  }

  async function cancelReservationAction() {
    if (!reservation.value) {
      setError('cancelReservation', 'Reservation is required.')
      return
    }

    reservation.value = await runTask('cancelReservation', () =>
      cancelReservation({
        reservation_id: reservation.value!.id,
        cancelled_at: new Date().toISOString(),
      }),
    )
    order.value = null
    payment.value = null
  }

  async function fetchOrderAction() {
    if (!order.value) {
      setError('fetchOrder', 'Order is required.')
      return
    }

    order.value = await runTask('fetchOrder', () => getOrder(order.value!.id))
  }

  async function loadMyOrders() {
    myOrders.value = await runTask('myOrders', listMyOrders)
  }

  function selectOrder(nextOrder: OrderResponse) {
    order.value = nextOrder
  }

  async function payOrderAction(input: { method: string; idempotencyKey: string }) {
    if (!order.value) {
      setError('payOrder', 'Order is required.')
      return
    }

    payment.value = await runTask('payOrder', () =>
      payOrder({
        order_id: order.value!.id,
        method: input.method,
      }, input.idempotencyKey),
    )
  }

  return {
    availability,
    cancelReservationAction,
    clearError,
    createOrderAction,
    currentUser,
    currentEvent,
    errors,
    events,
    fetchOrderAction,
    getError,
    isPending,
    joinQueueAction,
    loadEventBundle,
    loadEvents,
    loadMe,
    loginAction,
    loginDemoUser,
    logoutAction,
    loadMyOrders,
    myOrders,
    order,
    payment,
    purchaseToken,
    queueStatus,
    queueToken,
    refreshQueueStatus,
    refreshSections,
    registerAction,
    reservation,
    reserveTicketAction,
    resetCheckout,
    resetForEventChange,
    runTask,
    saleStatus,
    sections,
    selectEvent,
    selectOrder,
    selectedEventId,
    selectedSection,
    selectedSectionId,
    setError,
    userId,
    payOrderAction,
  }
})
