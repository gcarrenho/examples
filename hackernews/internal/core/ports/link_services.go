package ports

import domains "github.com/gcarrenho/hackernews/internal/core/domains/links"

type LinkSrv interface {
	CreateLink(link domains.Link) (domains.Link, error)
}
