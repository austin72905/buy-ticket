# Admin Backoffice

這份文件說明 `buy-ticket` 目前已實作的後台功能、權限邊界與資料一致性設計。

## 1. 後台範圍

目前後台包含：

- 獨立的 admin 登入、登出與 session。
- Admin user 與 organizer 管理。
- Event 與 section 查詢、建立和更新。
- 訂單查詢與敏感資料 reveal。
- Audit log 查詢。
- Organizer scope 權限隔離。
- Keyset pagination。
- Optimistic locking。

前台 `users` 與後台 `admin_users` 是不同帳號模型，session cookie 也分開管理。

## 2. 角色與資料範圍

目前支援兩種角色：

### `SUPER_ADMIN`

- 可管理所有 admin users 與 organizers。
- 可查看及管理所有 organizer 的 events 與 sections。
- 可查看所有訂單與 audit logs。
- 可使用 reveal API 查看訂單的完整使用者姓名與 email。

### `EVENT_ADMIN`

- 必須綁定一個 `organizer_id`。
- 只能查看與管理自己 organizer 的 events 與 sections。
- 訂單列表會自動限制在自己 organizer 的活動。
- 只能查看自己產生的 audit logs。
- 不能管理 admin users 或 organizers。
- 不能使用敏感資料 reveal API。

Organizer scope 由 service 與 repository 查詢條件強制套用，不依賴前端自行過濾。

## 3. Authentication API

| Method | Path | 說明 |
| --- | --- | --- |
| `POST` | `/admin/auth/login` | Admin 登入並建立 `buy_ticket_admin_session` cookie |
| `POST` | `/admin/auth/logout` | 清除目前 admin session |
| `GET` | `/admin/me` | 取得目前登入的 admin user |

除 login 外，所有 `/admin` API 都需要有效的 admin session。Disabled admin user 無法作為有效登入身分使用。

## 4. Admin User 與 Organizer API

以下 API 只允許 `SUPER_ADMIN` 使用：

| Method | Path | 說明 |
| --- | --- | --- |
| `GET` | `/admin/users` | 列出 admin users |
| `POST` | `/admin/users` | 建立 `SUPER_ADMIN` 或 `EVENT_ADMIN` |
| `PATCH` | `/admin/users/{adminUserId}` | 更新帳號、角色、狀態、密碼或 organizer |
| `GET` | `/admin/organizers` | 列出 organizers |
| `POST` | `/admin/organizers` | 建立 organizer |
| `PATCH` | `/admin/organizers/{organizerId}` | 更新 organizer 名稱或狀態 |

`EVENT_ADMIN` 必須綁定 organizer；`SUPER_ADMIN` 不綁定 organizer。系統也不允許 admin 將自己的帳號設為 disabled。

## 5. Event 與 Section API

`SUPER_ADMIN` 可以操作所有資料；`EVENT_ADMIN` 只限自己的 organizer。

| Method | Path | 說明 |
| --- | --- | --- |
| `GET` | `/admin/events` | 列出可存取的 events |
| `POST` | `/admin/events` | 建立 event |
| `GET` | `/admin/events/{eventId}` | 取得 event |
| `PATCH` | `/admin/events/{eventId}` | 更新 event |
| `GET` | `/admin/events/{eventId}/sections` | 列出 event sections |
| `POST` | `/admin/events/{eventId}/sections` | 建立 section |
| `PATCH` | `/admin/events/{eventId}/sections/{sectionId}` | 更新 section |

Event request 的 `status` 目前使用數字 enum。更新 event 時不能手動指定 `ON_SALE`；開售狀態由 event status scheduler 依 `PUBLISHED` 與售票時間推進。

Section 已支援：

- 價格。
- 總庫存、保留庫存、已售庫存與可用庫存。
- `purchase_limit`。
- `ACTIVE`、`INACTIVE`、`SOLD_OUT` 狀態。

更新 section 時，`total_quantity` 不得小於目前 `reserved_quantity + sold_quantity`，而 `purchase_limit` 不得超過總庫存。

## 6. 訂單列表與敏感資料

### 訂單列表

```http
GET /admin/orders
```

支援 query parameters：

- `event_id`
- `user_id`
- `status`
- `cursor_created_at`
- `cursor_id`
- `limit`

回應使用 keyset pagination：

```json
{
  "items": [],
  "next_cursor": {
    "created_at": "2026-06-28T12:00:00Z",
    "id": 12345
  }
}
```

排序鍵為 `created_at DESC, id DESC`。預設每頁 20 筆，server 會限制請求大小。一般訂單列表會遮罩：

- `user_name`
- `user_email`

### Reveal sensitive data

只有 `SUPER_ADMIN` 可以呼叫：

```http
POST /admin/orders/{orderId}/reveal-sensitive
```

Request：

```json
{
  "reason": "客服核對訂單"
}
```

`reason` 必填。成功後回傳完整姓名與 email，並寫入 `REVEAL_ORDER_SENSITIVE` audit log，包含 admin user、target order、reason、IP 與 user agent。

目前沒有 `GET /admin/orders/{orderId}` endpoint；完整敏感資料只能透過 reveal API 取得。

## 7. Audit Log

```http
GET /admin/audit-logs
```

Audit log 使用和訂單列表相同的 `cursor_created_at + cursor_id` keyset pagination。IP address 在 API response 中會遮罩。

目前會記錄以下操作：

| Action | Target |
| --- | --- |
| `ADMIN_USER_UPDATE` | `ADMIN_USER` |
| `ORGANIZER_UPDATE` | `ORGANIZER` |
| `EVENT_UPDATE` | `EVENT` |
| `SECTION_UPDATE` | `SECTION` |
| `REVEAL_ORDER_SENSITIVE` | `ORDER` |

`SUPER_ADMIN` 可以查看所有 logs；`EVENT_ADMIN` 目前只能查看自己產生的 logs。

## 8. Optimistic Locking

以下資料表具有 `version` 欄位：

- `admin_users`
- `organizers`
- `events`
- `event_sections`

所有 PATCH request 都必須帶 `expected_version`：

```json
{
  "expected_version": 3,
  "name": "更新後名稱"
}
```

更新 SQL 會使用目前 version 作為條件，成功後執行 `version = version + 1`。若資料已被其他 request 更新，API 回傳 `409 Conflict`，前端應重新讀取最新資料後再提交。

## 9. 狀態表示

API response 的 status 使用字串，例如：

- Admin user、organizer：`ACTIVE`、`DISABLED`
- Event：`DRAFT`、`PUBLISHED`、`ON_SALE`、`ENDED`
- Section：`ACTIVE`、`INACTIVE`、`SOLD_OUT`
- Order：`PENDING_PAYMENT`、`PAID`、`EXPIRED`、`CANCELLED`

Admin create／update request 的 status 目前仍使用數字 enum；詳細對照請參考 `doc/api-status-values.md`。

## 10. 目前限制

- 尚未提供獨立的單筆訂單詳情 API。
- Create 操作目前沒有全部寫入 audit log；主要記錄 update 與敏感資料 reveal。
- Events 與 sections 列表目前不是 keyset pagination。
- 尚未實作搶票請求紀錄、庫存流水或完整營運報表。
- 後台目前是 API 能力，若要作為完整營運工具，仍需由 frontend 補齊對應管理介面與操作流程。
