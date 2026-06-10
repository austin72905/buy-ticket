export interface EventResponse {
  id: number
  name: string
  venue: string
  status: number
  start_at: string
  end_at: string
  sale_start_at: string
  sale_end_at: string
  created_at: string
  updated_at: string
}

export interface SectionResponse {
  id: number
  event_id: number
  name: string
  price: number
  total_quantity: number
  reserved_quantity: number
  sold_quantity: number
  purchase_limit: number
  status: number
  created_at: string
  updated_at: string
}

export interface SectionAvailabilityResponse {
  section_id: number
  name: string
  price: number
  available_quantity: number
  reserved_quantity: number
  sold_quantity: number
  status: number
}

export interface SaleStatusResponse {
  event_id: number
  event_status: number
  is_on_sale: boolean
  queue_enabled: boolean
  can_join_queue: boolean
  can_reserve: boolean
  sale_start_at: string
  sale_end_at: string
  server_time: string
}

export interface JoinQueueRequest {
  event_id: number
  user_id: number
  client_id: string
  request_id: string
  channel: string
  access_code?: string
}

export interface QueueStatusResponse {
  queue_token: string
  status: number
  event_id: number
  user_id: number
  queue_position: number
  ahead_count: number
  estimated_wait_seconds: number
  purchase_token?: string
  purchase_token_expires_at?: string
  joined_at: string
  expired_at: string
  updated_at?: string
}

export interface ReserveTicketRequest {
  user_id: number
  event_id: number
  section_id: number
  quantity: number
  hold_until: string
  purchase_token: string
}

export interface ReservationResponse {
  id: number
  event_id: number
  section_id: number
  user_id: number
  quantity: number
  unit_price: number
  total_amount: number
  status: number
  expires_at: string
  created_at: string
  updated_at: string
}

export interface CreateOrderRequest {
  reservation_id: number
  order_no: string
  expires_at: string
}

export interface OrderResponse {
  id: number
  order_no: string
  user_id: number
  event_id: number
  section_id: number
  reservation_id: number
  quantity: number
  unit_price: number
  total_amount: number
  status: number
  expires_at: string
  created_at: string
  updated_at: string
}

export interface PayOrderRequest {
  order_id: number
  payment_no: string
  method: string
  amount: number
  paid_at: string
}

export interface PaymentResponse {
  id: number
  order_id: number
  payment_no: string
  method: string
  amount: number
  status: number
  paid_at?: string
  failed_at?: string
  created_at: string
  updated_at: string
}
