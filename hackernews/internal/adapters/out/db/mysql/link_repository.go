package mysql

import (
	"github.com/gcarrenho/hackernews/internal/adapters/in/graph/model"
	"github.com/gcarrenho/hackernews/internal/core/ports"
)

var _ ports.LinkRepository = (*LinkRepositoryImpl)(nil)

type LinkRepositoryImpl struct {
}

func NewLinkRepository() *LinkRepositoryImpl {
	return &LinkRepositoryImpl{}
}

func (l *LinkRepositoryImpl) InsertLink(link model.Link) error {
	return nil
}
