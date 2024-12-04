package domains

import domains "github.com/gcarrenho/hackernews/internal/core/domains/users"

type Link struct {
	ID      string
	Title   string
	Address string
	User    *domains.User
}
