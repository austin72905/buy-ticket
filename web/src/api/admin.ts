import http from '../lib/http'
import type {
  AdminAuditLogListParams,
  AdminAuditLogListResponse,
  AdminEventResponse,
  AdminLoginRequest,
  AdminOrderListParams,
  AdminOrderListResponse,
  AdminOrderSensitiveResponse,
  AdminSectionResponse,
  AdminUserResponse,
  CreateAdminEventRequest,
  CreateAdminSectionRequest,
  CreateAdminUserRequest,
  CreateOrganizerRequest,
  OrganizerResponse,
  RevealSensitiveRequest,
} from '../types/admin'

export async function adminLogin(payload: AdminLoginRequest) {
  const { data } = await http.post<AdminUserResponse>('/admin/auth/login', payload)
  return data
}

export async function adminLogout() {
  await http.post('/admin/auth/logout')
}

export async function getAdminMe() {
  const { data } = await http.get<AdminUserResponse>('/admin/me')
  return data
}

export async function listAdminUsers() {
  const { data } = await http.get<AdminUserResponse[]>('/admin/users')
  return data
}

export async function createAdminUser(payload: CreateAdminUserRequest) {
  const { data } = await http.post<AdminUserResponse>('/admin/users', payload)
  return data
}

export async function listOrganizers() {
  const { data } = await http.get<OrganizerResponse[]>('/admin/organizers')
  return data
}

export async function createOrganizer(payload: CreateOrganizerRequest) {
  const { data } = await http.post<OrganizerResponse>('/admin/organizers', payload)
  return data
}

export async function listAdminOrders(params: AdminOrderListParams) {
  const { data } = await http.get<AdminOrderListResponse>('/admin/orders', { params })
  return data
}

export async function revealAdminOrderSensitive(orderId: number, payload: RevealSensitiveRequest) {
  const { data } = await http.post<AdminOrderSensitiveResponse>(
    `/admin/orders/${orderId}/reveal-sensitive`,
    payload,
  )
  return data
}

export async function listAdminEvents() {
  const { data } = await http.get<AdminEventResponse[]>('/admin/events')
  return data
}

export async function createAdminEvent(payload: CreateAdminEventRequest) {
  const { data } = await http.post<AdminEventResponse>('/admin/events', payload)
  return data
}

export async function getAdminEvent(eventId: number) {
  const { data } = await http.get<AdminEventResponse>(`/admin/events/${eventId}`)
  return data
}

export async function listAdminEventSections(eventId: number) {
  const { data } = await http.get<AdminSectionResponse[]>(`/admin/events/${eventId}/sections`)
  return data
}

export async function createAdminEventSection(eventId: number, payload: CreateAdminSectionRequest) {
  const { data } = await http.post<AdminSectionResponse>(`/admin/events/${eventId}/sections`, payload)
  return data
}

export async function listAdminAuditLogs(params: AdminAuditLogListParams) {
  const { data } = await http.get<AdminAuditLogListResponse>('/admin/audit-logs', { params })
  return data
}
