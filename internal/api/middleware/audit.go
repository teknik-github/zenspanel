package middleware

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

// Audit returns Gin middleware that records every mutating request
// (POST/PUT/DELETE) to the audit_logs table after the handler runs. We skip
// GET because read traffic is high-volume and not interesting for audit;
// we skip 5xx because failed requests didn't actually change state.
//
// The middleware is best-effort — a Create() failure on the audit row is
// swallowed rather than turned into a 500 for the user, because the user's
// request already succeeded by the time we get here.
func Audit(logs *store.AuditLogStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		method := c.Request.Method
		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			return
		}
		if c.Writer.Status() >= 500 {
			return
		}

		entry := &store.AuditLog{
			Action:    method + " " + c.FullPath(),
			IPAddress: c.ClientIP(),
		}
		if uid := auth.GetUserID(c); uid > 0 {
			entry.UserID = sql.NullInt64{Int64: int64(uid), Valid: true}
		}
		if id := c.Param("id"); id != "" {
			entry.Resource = sql.NullString{String: id, Valid: true}
		}
		if ua := c.GetHeader("User-Agent"); ua != "" {
			entry.UserAgent = sql.NullString{String: ua, Valid: true}
		}
		_ = logs.Create(entry)
	}
}
