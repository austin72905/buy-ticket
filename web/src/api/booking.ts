import http from '../lib/http'
import type {
  CancelReservationRequest,
  CreateOrderRequest,
  EventResponse,
  ExpireReservationRequest,
  JoinQueueRequest,
  OrderResponse,
  PaymentResponse,
  PayOrderRequest,
  QueueStatusResponse,
  ReservationResponse,
  ReserveTicketRequest,
  SaleStatusResponse,
  SectionAvailabilityResponse,
  SectionResponse,
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

export async function payOrder(payload: PayOrderRequest) {
  const { data } = await http.post<PaymentResponse>('/payments', payload)
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

export async function listUserReservations(userId: number) {
  const { data } = await http.get<ReservationResponse[]>(`/users/${userId}/reservations`)
  return data
}

export async function listUserOrders(userId: number) {
  const { data } = await http.get<OrderResponse[]>(`/users/${userId}/orders`)
  return data
}

export async function listUserPayments(userId: number) {
  const { data } = await http.get<PaymentResponse[]>(`/users/${userId}/payments`)
  return data
}
