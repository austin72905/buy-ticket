import http from '../lib/http'
import type {
  CreateOrderRequest,
  EventResponse,
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
