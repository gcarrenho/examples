package ports

import (
	"context"

	"github.com/gcarrenho/routeguide/internal/core/model"
)

type FeatureRepository interface {
	GetFeature(ctx context.Context, point model.Point) (*model.Feature, error)
	ListFeatures(ctx context.Context, rect model.Rectangle) ([]model.Feature, error)
}
