// Package auth is identity for the whole backend: roles, tokens, passwords, OTP codes,
// Google ID tokens and the middleware levels (BACKEND_PLAN.md section 5).
package auth

// Role mirrors the frontend UserRole enum (lib/features/auth/domain/entities/user_role.dart).
type Role string

// The five roles, named exactly as the Dart enum values.
const (
	RoleReader         Role = "reader"
	RoleModerator      Role = "moderator"
	RoleCatalogManager Role = "catalogManager"
	RoleSupport        Role = "support"
	RoleSuperAdmin     Role = "superAdmin"
)

// Perm is one staff permission.
type Perm string

// Permissions behind the staff:<perm> auth levels.
const (
	PermModerate Perm = "moderate"
	PermCatalog  Perm = "catalog"
	PermOrders   Perm = "orders"
)

// ParseRole checks a role name from the database or a command line.
func ParseRole(s string) (Role, bool) {
	switch r := Role(s); r {
	case RoleReader, RoleModerator, RoleCatalogManager, RoleSupport, RoleSuperAdmin:
		return r, true
	}
	return "", false
}

// IsStaff: staff can open the admin area.
func (r Role) IsStaff() bool { return r != RoleReader }

// CanModerate: approve used-book listings and handle reports.
func (r Role) CanModerate() bool { return r == RoleModerator || r == RoleSuperAdmin }

// CanManageCatalog: add and edit books, stock, collections and banners.
func (r Role) CanManageCatalog() bool { return r == RoleCatalogManager || r == RoleSuperAdmin }

// CanManageOrders: handle orders, returns and refunds.
func (r Role) CanManageOrders() bool { return r == RoleSupport || r == RoleSuperAdmin }

// Can reports whether the role holds a permission.
func (r Role) Can(p Perm) bool {
	switch p {
	case PermModerate:
		return r.CanModerate()
	case PermCatalog:
		return r.CanManageCatalog()
	case PermOrders:
		return r.CanManageOrders()
	}
	return false
}
