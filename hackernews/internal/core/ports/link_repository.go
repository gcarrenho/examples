package ports

import "github.com/gcarrenho/hackernews/internal/adapters/in/graph/model"

type LinkRepository interface {
	InsertLink(link model.Link) error
}
