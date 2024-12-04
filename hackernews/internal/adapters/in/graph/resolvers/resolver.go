package resolvers

import "github.com/gcarrenho/hackernews/internal/core/ports"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	LinkSrv ports.LinkSrv
}
