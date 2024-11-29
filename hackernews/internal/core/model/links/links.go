package links

import "github.com/gcarrenho/hackernews/internal/core/model/users"

type Link struct {
	ID      string
	Title   string
	Address string
	User    *users.User
}
