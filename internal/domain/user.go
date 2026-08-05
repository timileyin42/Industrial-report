package domain

// Role gates what a user can see: an operator sees the whole fleet, a
// restricted user is scoped to exactly one site. Enforced server-side on
// every API call — see CLAUDE.md "Access control".
type Role string

const (
	RoleOperator   Role = "operator"
	RoleRestricted Role = "restricted"
)

type User struct {
	ID     int64
	Email  string
	Role   Role
	SiteID *string // nil for operator, required for restricted
}
