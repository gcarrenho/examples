package services

import (
	domains "github.com/gcarrenho/hackernews/internal/core/domains/links"
	"github.com/gcarrenho/hackernews/internal/core/ports"
)

var _ ports.LinkSrv = (*LinkSrvImpl)(nil)

type LinkSrvImpl struct {
	linkRepository ports.LinkRepository
}

func NewLinkSrv(linkRepository ports.LinkRepository) *LinkSrvImpl {
	return &LinkSrvImpl{
		linkRepository: linkRepository,
	}
}

func (l *LinkSrvImpl) CreateLink(link domains.Link) (domains.Link, error) {
	return link, nil
}
