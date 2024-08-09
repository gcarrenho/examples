package service

import (
	"context"
	"math"
	"time"

	"github.com/gcarrenho/routeguide2/internal/core/model"
	"github.com/gcarrenho/routeguide2/internal/core/ports"
)

var _ ports.FeatureService = (*FeatureServiceImpl)(nil)

type FeatureServiceImpl struct {
	featureRepo ports.FeatureRepository
}

func NewFeatureService(repo ports.FeatureRepository) ports.FeatureService {
	return &FeatureServiceImpl{featureRepo: repo}
}

func (s *FeatureServiceImpl) GetFeature(ctx context.Context, point model.Point) (*model.Feature, error) {
	return s.featureRepo.GetFeature(ctx, point)
}

func (s *FeatureServiceImpl) ListFeatures(ctx context.Context, rect model.Rectangle) ([]model.Feature, error) {
	return s.featureRepo.ListFeatures(ctx, rect)
}

func (s *FeatureServiceImpl) RecordRoute(ctx context.Context, points []model.Point) (*model.RouteSummary, error) {
	var pointCount, featureCount, distance int32
	var lastPoint *model.Point
	startTime := time.Now()

	for _, point := range points {
		pointCount++
		feature, _ := s.featureRepo.GetFeature(ctx, point)
		if feature != nil {
			featureCount++
		}
		if lastPoint != nil {
			distance += calcDistance(*lastPoint, point)
		}
		lastPoint = &point
	}

	endTime := time.Now()
	return &model.RouteSummary{
		PointCount:   pointCount,
		FeatureCount: featureCount,
		Distance:     distance,
		ElapsedTime:  int32(endTime.Sub(startTime).Seconds()),
	}, nil
}

func calcDistance(p1, p2 model.Point) int32 {
	const CordFactor float64 = 1e7
	const R = float64(6371000) // earth radius in metres
	lat1 := toRadians(float64(p1.Latitude) / CordFactor)
	lat2 := toRadians(float64(p2.Latitude) / CordFactor)
	lng1 := toRadians(float64(p1.Longitude) / CordFactor)
	lng2 := toRadians(float64(p2.Longitude) / CordFactor)
	dlat := lat2 - lat1
	dlng := lng2 - lng1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c
	return int32(distance)
}

func toRadians(num float64) float64 {
	return num * math.Pi / 180
}
