package domain

import "time"

type AdminRole string

const (
	AdminRoleSuperAdmin AdminRole = "SUPER_ADMIN"
	AdminRoleEventAdmin AdminRole = "EVENT_ADMIN"
)

type AdminUserStatus int8

const (
	AdminUserStatusActive AdminUserStatus = iota + 1
	AdminUserStatusDisabled
)

type OrganizerStatus int8

const (
	OrganizerStatusActive OrganizerStatus = iota + 1
	OrganizerStatusDisabled
)

type Organizer struct {
	ID        int64
	Name      string
	Status    OrganizerStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AdminUser struct {
	ID           int64
	OrganizerID  *int64
	Name         string
	Email        string
	PasswordHash string
	Role         AdminRole
	Status       AdminUserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (a AdminUser) IsActive() bool {
	return a.Status == AdminUserStatusActive
}

func (a AdminUser) IsSuperAdmin() bool {
	return a.Role == AdminRoleSuperAdmin
}

func (a AdminUser) IsEventAdmin() bool {
	return a.Role == AdminRoleEventAdmin
}

type AdminAuditLog struct {
	ID          int64
	AdminUserID int64
	AdminUser   *AdminUser
	Action      string
	TargetType  string
	TargetID    int64
	TargetName  *string
	Reason      *string
	IPAddress   *string
	UserAgent   *string
	CreatedAt   time.Time
}

type AdminAuditLogListCursor struct {
	CreatedAt *time.Time
	ID        int64
}

type AdminAuditLogListFilter struct {
	AdminUserID int64
	Cursor      AdminAuditLogListCursor
	Limit       int
}

type AdminOrder struct {
	ID            int64
	OrderNo       string
	ReservationID int64
	EventID       int64
	EventName     string
	SectionID     int64
	SectionName   string
	UserID        int64
	UserName      string
	UserEmail     string
	Quantity      int
	UnitPrice     int64
	TotalAmount   int64
	Status        OrderStatus
	ExpiresAt     time.Time
	PaidAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AdminOrderListCursor struct {
	CreatedAt *time.Time
	ID        int64
}

type AdminOrderListFilter struct {
	OrganizerID *int64
	EventID     int64
	UserID      int64
	Status      OrderStatus
	Cursor      AdminOrderListCursor
	Limit       int
}
