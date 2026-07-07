# API Status Values

這份文件給前端對齊 API response 的 `status` 字串值。

## 設計規則

- API response 的 `status` 回傳字串，不再回數字。
- DB 與後端 domain 內部仍可維持 enum number。
- 前端顯示、判斷流程、狀態 badge 應以字串為準。
- 後台建立 event / section 的 request 目前仍先接受數字 `status`，後續可再獨立改成字串 input。

## Event Status

| Status | 說明 |
| --- | --- |
| `DRAFT` | 草稿 |
| `PUBLISHED` | 已上架 |
| `ON_SALE` | 開售中 |
| `ENDED` | 已結束 |

使用位置：

- `GET /events`
- `GET /events/{eventId}`
- `GET /sale/status` 的 `event_status`
- `GET /admin/events`
- `GET /admin/events/{eventId}`

## Section Status

| Status | 說明 |
| --- | --- |
| `ACTIVE` | 可販售 |
| `INACTIVE` | 停用 |
| `SOLD_OUT` | 售完 |

使用位置：

- `GET /events/{eventId}/sections`
- `GET /events/{eventId}/availability`
- `GET /admin/events/{eventId}/sections`

## Queue Status

| Status | 說明 |
| --- | --- |
| `WAITING` | 排隊中 |
| `READY` | 已放行，可取得 purchase token |
| `EXPIRED` | queue token 已過期 |

使用位置：

- `POST /queue/join`
- `GET /queue/status/{queueToken}`

## Reservation Status

| Status | 說明 |
| --- | --- |
| `HOLDING` | 已保留，尚未確認付款 |
| `CONFIRMED` | 已付款確認 |
| `EXPIRED` | 已逾期 |
| `CANCELLED` | 已取消 |

使用位置：

- `POST /reservations`
- `GET /reservations/{reservationId}`
- `GET /me/reservations`

## Order Status

| Status | 說明 |
| --- | --- |
| `PENDING_PAYMENT` | 待付款 |
| `PAID` | 已付款 |
| `EXPIRED` | 已逾期 |
| `CANCELLED` | 已取消 |

使用位置：

- `POST /orders`
- `GET /orders/{orderId}`
- `GET /orders/order-no/{orderNo}`
- `GET /me/orders`
- `GET /admin/orders`

## Payment Status

| Status | 說明 |
| --- | --- |
| `PENDING` | 待付款 |
| `PAID` | 已付款 |
| `FAILED` | 付款失敗 |
| `REFUNDED` | 已退款 |

使用位置：

- `POST /payments`
- `GET /payments/{paymentNo}`
- `GET /me/payments`

## Payment Attempt Status

| Status | 說明 |
| --- | --- |
| `PROCESSING` | 已建立付款嘗試，等待 provider callback |
| `SUCCEEDED` | provider callback 成功 |
| `FAILED` | provider 回覆付款失敗 |
| `TIMEOUT` | provider timeout 或 circuit breaker open |
| `CANCELLED` | 付款嘗試已取消 |

使用位置：

- `POST /payments/start`

## Admin Status

Admin user:

| Status | 說明 |
| --- | --- |
| `ACTIVE` | 啟用 |
| `DISABLED` | 停用 |

Organizer:

| Status | 說明 |
| --- | --- |
| `ACTIVE` | 啟用 |
| `DISABLED` | 停用 |

使用位置：

- `GET /admin/users`
- `GET /admin/organizers`

## 前端處理建議

前端不要再用數字判斷狀態：

```ts
if (order.status === 'PENDING_PAYMENT') {
  // show payment button
}
```

狀態 badge 可用 map：

```ts
const orderStatusLabel: Record<string, string> = {
  PENDING_PAYMENT: '待付款',
  PAID: '已付款',
  EXPIRED: '已逾期',
  CANCELLED: '已取消',
}
```
