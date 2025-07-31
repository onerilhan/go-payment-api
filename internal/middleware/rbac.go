package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/auth"
	"github.com/onerilhan/go-payment-api/internal/middleware/errors"
)

// Permission represents a specific permission
type Permission string

// Define available permissions
const (
	// User permissions
	PermViewOwnProfile   Permission = "view_own_profile"
	PermUpdateOwnProfile Permission = "update_own_profile"
	PermDeleteOwnProfile Permission = "delete_own_profile"
	PermViewOwnBalance   Permission = "view_own_balance"
	PermMakeTransaction  Permission = "make_transaction"

	// Admin permissions
	PermViewAllUsers        Permission = "view_all_users"
	PermViewAnyUser         Permission = "view_any_user"
	PermUpdateAnyUser       Permission = "update_any_user"
	PermDeleteAnyUser       Permission = "delete_any_user"
	PermViewAllBalances     Permission = "view_all_balances"
	PermViewAnyBalance      Permission = "view_any_balance"
	PermViewAllTransactions Permission = "view_all_transactions"
	PermSystemManagement    Permission = "system_management"

	// Moderator permissions
	PermViewUserList     Permission = "view_user_list"
	PermViewUserDetails  Permission = "view_user_details"
	PermModerateUsers    Permission = "moderate_users"
	PermViewTransactions Permission = "view_transactions"
)

// RolePermissions defines permissions for each role
var RolePermissions = map[string][]Permission{
	"user": {
		PermViewOwnProfile,
		PermUpdateOwnProfile,
		PermDeleteOwnProfile,
		PermViewOwnBalance,
		PermMakeTransaction,
	},
	"mod": {
		// Moderator inherits user permissions
		PermViewOwnProfile,
		PermUpdateOwnProfile,
		PermDeleteOwnProfile,
		PermViewOwnBalance,
		PermMakeTransaction,
		// Plus moderator-specific permissions
		PermViewUserList,
		PermViewUserDetails,
		PermModerateUsers,
		PermViewTransactions,
	},
	"admin": {
		// Admin has all permissions
		PermViewOwnProfile,
		PermUpdateOwnProfile,
		PermDeleteOwnProfile,
		PermViewOwnBalance,
		PermMakeTransaction,
		PermViewAllUsers,
		PermViewAnyUser,
		PermUpdateAnyUser,
		PermDeleteAnyUser,
		PermViewAllBalances,
		PermViewAnyBalance,
		PermViewAllTransactions,
		PermSystemManagement,
		PermViewUserList,
		PermViewUserDetails,
		PermModerateUsers,
		PermViewTransactions,
	},
}

// ResourceOwnership checks if user owns the resource
type ResourceOwnership func(userID int, r *http.Request) bool

// RBACConfig RBAC middleware configuration
type RBACConfig struct {
	RequiredPermission Permission
	ResourceOwnership  ResourceOwnership // Optional: Check if user owns the resource
	AllowOwner         bool              // Allow resource owner even without permission
}

// RequirePermission creates RBAC middleware for specific permission
func RequirePermission(permission Permission) func(http.Handler) http.Handler {
	return RequirePermissionWithConfig(&RBACConfig{
		RequiredPermission: permission,
		AllowOwner:         false,
	})
}

// RequirePermissionWithOwnership creates RBAC middleware with resource ownership check
func RequirePermissionWithOwnership(permission Permission, ownershipCheck ResourceOwnership) func(http.Handler) http.Handler {
	return RequirePermissionWithConfig(&RBACConfig{
		RequiredPermission: permission,
		ResourceOwnership:  ownershipCheck,
		AllowOwner:         true,
	})
}

// RequirePermissionWithConfig creates RBAC middleware with full config
func RequirePermissionWithConfig(config *RBACConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context (set by AuthMiddleware)
			claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
			if !ok {
				log.Error().
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("RBAC: User context not found - AuthMiddleware might be missing")

				panic(&errors.AuthError{
					Message:    "Authentication required",
					StatusCode: http.StatusUnauthorized,
				})
			}

			// Get user role from JWT claims
			userRole := getUserRole(claims)

			// Resource ownership check (if configured)
			if config.AllowOwner && config.ResourceOwnership != nil {
				if config.ResourceOwnership(claims.UserID, r) {
					log.Debug().
						Int("user_id", claims.UserID).
						Str("role", userRole).
						Str("permission", string(config.RequiredPermission)).
						Str("path", r.URL.Path).
						Msg("RBAC: Access granted - Resource owner")

					next.ServeHTTP(w, r)
					return
				}
			}

			// Permission check
			if !hasPermission(userRole, config.RequiredPermission) {
				log.Warn().
					Int("user_id", claims.UserID).
					Str("role", userRole).
					Str("required_permission", string(config.RequiredPermission)).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("RBAC: Access denied - Insufficient permissions")

				panic(&errors.RBACError{
					Message:    "Bu işlem için yetkiniz bulunmuyor",
					StatusCode: http.StatusForbidden,
					Resource:   r.URL.Path,
					Action:     r.Method,
				})
			}

			log.Debug().
				Int("user_id", claims.UserID).
				Str("role", userRole).
				Str("permission", string(config.RequiredPermission)).
				Str("path", r.URL.Path).
				Msg("RBAC: Access granted - Permission granted")

			// Permission granted, continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// hasPermission checks if role has the required permission
func hasPermission(role string, permission Permission) bool {
	permissions, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// getUserRole extracts user role from JWT claims
func getUserRole(claims *auth.Claims) string {
	// JWT'den role'u al
	if claims.Role != "" {
		return claims.Role
	}

	// Fallback: Default role assignment (should not happen in production)
	log.Warn().
		Int("user_id", claims.UserID).
		Msg("Role not found in JWT claims, using default 'user' role")

	return "user"
}

// Common resource ownership checkers

// UserResourceOwnership checks if user owns user resource (/users/{id})
func UserResourceOwnership(userID int, r *http.Request) bool {
	vars := mux.Vars(r)
	resourceIDStr, exists := vars["id"]
	if !exists {
		return false
	}

	resourceID, err := strconv.Atoi(resourceIDStr)
	if err != nil {
		return false
	}

	return userID == resourceID
}

// TransactionResourceOwnership checks if user owns transaction resource
func TransactionResourceOwnership(userID int, r *http.Request) bool {
	// For transaction endpoints, we might need to query database
	// to check if user owns the transaction
	// For now, we'll implement a simple check

	vars := mux.Vars(r)
	transactionIDStr, exists := vars["id"]
	if !exists {
		// For endpoints without ID (like /transactions/history), allow if it's the user's own data
		return true
	}

	// In real implementation, we'd query database to check transaction ownership
	// For now, we'll allow access (actual ownership check would be in service layer)
	_ = transactionIDStr
	return true
}

// Convenience middleware functions for common use cases

// RequireAdmin requires admin role
func RequireAdmin() func(http.Handler) http.Handler {
	return RequirePermission(PermSystemManagement)
}

// RequireAdminOrMod requires admin or moderator role
func RequireAdminOrMod() func(http.Handler) http.Handler {
	return RequirePermission(PermViewUserList)
}

// RequireOwnershipOrAdmin allows resource owner or admin
func RequireOwnershipOrAdmin(ownershipCheck ResourceOwnership) func(http.Handler) http.Handler {
	return RequirePermissionWithOwnership(PermViewAnyUser, ownershipCheck)
}

// Endpoint-specific middleware functions

// UserManagementRBAC RBAC for user management endpoints
func UserManagementRBAC() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Different permissions based on the endpoint
			path := r.URL.Path
			method := r.Method

			var config *RBACConfig

			switch {
			case strings.Contains(path, "/users") && method == "GET":
				if strings.Contains(path, "/profile") {
					// Own profile access
					config = &RBACConfig{
						RequiredPermission: PermViewOwnProfile,
						AllowOwner:         false,
					}
				} else if mux.Vars(r)["id"] != "" {
					// Specific user access - allow owner or admin/mod
					config = &RBACConfig{
						RequiredPermission: PermViewAnyUser,
						ResourceOwnership:  UserResourceOwnership,
						AllowOwner:         true,
					}
				} else {
					// List all users - requires admin/mod
					config = &RBACConfig{
						RequiredPermission: PermViewUserList,
						AllowOwner:         false,
					}
				}

			case strings.Contains(path, "/users") && method == "PUT":
				// Update user - allow owner or admin
				config = &RBACConfig{
					RequiredPermission: PermUpdateAnyUser,
					ResourceOwnership:  UserResourceOwnership,
					AllowOwner:         true,
				}

			case strings.Contains(path, "/users") && method == "DELETE":
				// Delete user - allow owner or admin
				config = &RBACConfig{
					RequiredPermission: PermDeleteAnyUser,
					ResourceOwnership:  UserResourceOwnership,
					AllowOwner:         true,
				}

			default:
				// Default: require user permission
				config = &RBACConfig{
					RequiredPermission: PermViewOwnProfile,
					AllowOwner:         false,
				}
			}

			// Apply the RBAC check
			rbacHandler := RequirePermissionWithConfig(config)
			rbacHandler(next).ServeHTTP(w, r)
		})
	}
}
