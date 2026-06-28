export interface AdminLoginRequest {
  email: string
  password: string
}

export interface AdminUserResponse {
  id: number
  name: string
  email: string
  role: string
  status: number
  organizer_id?: number
  created_at: string
  updated_at: string
}

export interface AdminCursor {
  created_at: string
  id: number
}

export interface AdminOrderResponse {
  id: number
  order_no: string
  user_id: number
  user_name: string
  user_email: string
  event_id: number
  event_name: string
  section_id: number
  section_name: string
  reservation_id: number
  quantity: number
  unit_price: number
  total_amount: number
  status: number
  expires_at: string
  paid_at?: string
  created_at: string
  updated_at: string
}

export interface AdminOrderListResponse {
  items: AdminOrderResponse[]
  next_cursor?: AdminCursor
}

export interface AdminOrderListParams {
  limit?: number
  cursor_created_at?: string
  cursor_id?: number
  event_id?: number
  user_id?: number
  status?: number
}

export interface AdminOrderFilters {
  eventId: string
  userId: string
  status: string
}

export interface RevealSensitiveRequest {
  reason: string
}

export interface AdminOrderSensitiveResponse {
  order_id: number
  order_no: string
  user_id: number
  user_name: string
  user_email: string
}

export interface AdminEventResponse {
  id: number
  name: string
  venue: string
  organizer_id: number
  status: number
  start_at: string
  end_at: string
  sale_start_at: string
  sale_end_at: string
  created_at: string
  updated_at: string
}

export interface AdminSectionResponse {
  id: number
  event_id: number
  name: string
  price: number
  total_quantity: number
  reserved_quantity: number
  sold_quantity: number
  available_quantity: number
  purchase_limit: number
  status: number
  created_at: string
  updated_at: string
}

export interface AdminAuditLogResponse {
  id: number
  admin_user_id: number
  action: string
  target_type: string
  target_id: number
  reason?: string
  ip_address?: string
  user_agent?: string
  created_at: string
}

export interface AdminAuditLogListResponse {
  items: AdminAuditLogResponse[]
  next_cursor?: AdminCursor
}

export interface AdminAuditLogListParams {
  limit?: number
  cursor_created_at?: string
  cursor_id?: number
}
