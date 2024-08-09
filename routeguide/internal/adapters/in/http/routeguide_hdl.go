package hdl

import (
	"context"
	"errors"
	"net/http"

	"github.com/gcarrenho/routeguide2/internal/core/model"
	"github.com/gcarrenho/routeguide2/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type RouteGuideHdl struct {
	rg         *gin.RouterGroup
	featureSvc ports.FeatureService
}

func NewRouteGUideHdl(rg *gin.RouterGroup, featureSvc ports.FeatureService) {
	hdl := RouteGuideHdl{
		rg:         rg,
		featureSvc: featureSvc,
	}

	rg.GET("/", hdl.getFeature)
}

func (hdl RouteGuideHdl) getFeature(c *gin.Context) {
	point := model.Point{
		Latitude:  409146138,
		Longitude: -746188906,
	}

	feature, err := hdl.featureSvc.GetFeature(context.Background(), point)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, errors.New("Fail"))
		return
	}

	c.JSON(http.StatusOK, feature)
}
