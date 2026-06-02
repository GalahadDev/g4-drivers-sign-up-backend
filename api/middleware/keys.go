package middleware

type contextKey int

const (
	ContextKeyUserID   contextKey = iota
	ContextKeyUserRole
)
