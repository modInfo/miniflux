// Copyright 2017 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package middleware

import (
	"context"
	"net/http"

	"github.com/miniflux/miniflux/logger"
	"github.com/miniflux/miniflux/storage"
)

// BasicAuthMiddleware is the middleware for HTTP Basic authentication.
type BasicAuthMiddleware struct {
	store *storage.Storage
}

// Handler executes the middleware.
func (b *BasicAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		errorResponse := `{"error_message": "Not Authorized"}`

		username, password, authOK := r.BasicAuth()
		if !authOK {
			logger.Debug("[Middleware:BasicAuth] No authentication headers sent")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(errorResponse))
			return
		}

		if err := b.store.CheckPassword(username, password); err != nil {
			logger.Info("[Middleware:BasicAuth] Invalid username or password: %s", username)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(errorResponse))
			return
		}

		user, err := b.store.UserByUsername(username)
		if err != nil {
			logger.Error("[Middleware:BasicAuth] %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errorResponse))
			return
		}

		if user == nil {
			logger.Info("[Middleware:BasicAuth] User not found: %s", username)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(errorResponse))
			return
		}

		logger.Info("[Middleware:BasicAuth] User authenticated: %s", username)
		b.store.SetLastLogin(user.ID)

		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDContextKey, user.ID)
		ctx = context.WithValue(ctx, UserTimezoneContextKey, user.Timezone)
		ctx = context.WithValue(ctx, IsAdminUserContextKey, user.IsAdmin)
		ctx = context.WithValue(ctx, IsAuthenticatedContextKey, true)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// NewBasicAuthMiddleware returns a new BasicAuthMiddleware.
func NewBasicAuthMiddleware(s *storage.Storage) *BasicAuthMiddleware {
	return &BasicAuthMiddleware{store: s}
}
