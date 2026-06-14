import http from '../lib/http'
import type {
  CancelReservationRequest,
  CreateOrderRequest,
  EventResponse,
  ExpireReservationRequest,
  JoinQueueRequest,
  LoginRequest,
  OrderResponse,
  PaymentResponse,
  PayOrderRequest,
  QueueStatusResponse,
  RegisterRequest,
  ReservationResponse,
  ReserveTicketRequest,
  SaleStatusResponse,
  SectionAvailabilityResponse,
  SectionResponse,
  UserResponse,
} from '../types/api'

export async function listEvents() {
  const { data } = await http.get<EventResponse[]>('/events')
  return data
}

export async function getEvent(eventId: number) {
  const { data } = await http.get<EventResponse>(`/events/${eventId}`)
  return data
}

export async function getSections(eventId: number) {
  const { data } = await http.get<SectionResponse[]>(`/events/${eventId}/sections`)
  return data
}

export async function getAvailability(eventId: number) {
  const { data } = await http.get<SectionAvailabilityResponse[]>(`/events/${eventId}/availability`)
  return data
}

export async function getSaleStatus(eventId: number) {
  const { data } = await http.get<SaleStatusResponse>('/sale/status', {
    params: { event_id: eventId },
  })
  return data
}

export async function register(payload: RegisterRequest) {
  const { data } = await http.post<UserResponse>('/auth/register', payload)
  return data
}

export async function login(payload: LoginRequest) {
  const { data } = await http.post<UserResponse>('/auth/login', payload)
  return data
}

export async function logout() {
  await http.post('/auth/logout')
}

export async function getMe() {
  const { data } = await http.get<UserResponse>('/me')
  return data
}

export async function joinQueue(payload: JoinQueueRequest) {
  const { data } = await http.post<QueueStatusResponse>('/queue/join', payload)
  return data
}

export async function getQueueStatus(queueToken: string) {
  const { data } = await http.get<QueueStatusResponse>(`/queue/status/${queueToken}`)
  return data
}

export async function reserveTicket(payload: ReserveTicketRequest) {
  const { data } = await http.post<ReservationResponse>('/reservations', payload)
  return data
}

export async function getReservation(reservationId: number) {
  const { data } = await http.get<ReservationResponse>(`/reservations/${reservationId}`)
  return data
}

export async function cancelReservation(payload: CancelReservationRequest) {
  const { data } = await http.post<ReservationResponse>('/reservations/cancel', payload)
  return data
}

export async function expireReservation(payload: ExpireReservationRequest) {
  const { data } = await http.post<ReservationResponse>('/reservations/expire', payload)
  return data
}

export async function createOrder(payload: CreateOrderRequest) {
  const { data } = await http.post<OrderResponse>('/orders', payload)
  return data
}

export async function payOrder(payload: PayOrderRequest, idempotencyKey: string) {
  const { data } = await http.post<PaymentResponse>('/payments', payload, {
    headers: {
      'Idempotency-Key': idempotencyKey,
    },
  })
  return data
}

export async function getOrder(orderId: number) {
  const { data } = await http.get<OrderResponse>(`/orders/${orderId}`)
  return data
}

export async function getOrderByOrderNo(orderNo: string) {
  const { data } = await http.get<OrderResponse>(`/orders/order-no/${orderNo}`)
  return data
}

export async function getPaymentByPaymentNo(paymentNo: string) {
  const { data } = await http.get<PaymentResponse>(`/payments/${paymentNo}`)
  return data
}

export async function listMyReservations() {
  const { data } = await http.get<ReservationResponse[]>('/me/reservations')
  return data
}

export async function listMyOrders() {
  const { data } = await http.get<OrderResponse[]>('/me/orders')
  return data
}

export async function listMyPayments() {
  const { data } = await http.get<PaymentResponse[]>('/me/payments')
  return data
}
