package adapters

import (
	"context"
	"fmt"
	"io"
	"sync"

	api "github.com/gcarrenho/routeguide/api/v1"
	"github.com/gcarrenho/routeguide/internal/core/model"
	"github.com/gcarrenho/routeguide/internal/core/ports"
)

var _ api.RouteGuideServer = (*GRPCServer)(nil)

type GRPCServer struct {
	api.UnimplementedRouteGuideServer
	featureSvc ports.FeatureService
	mu         sync.Mutex // protects routeNotes
	routeNotes map[string][]*api.RouteNote
}

func NewGRPCServer(featureSvc ports.FeatureService) *GRPCServer {
	return &GRPCServer{
		featureSvc: featureSvc,
		routeNotes: make(map[string][]*api.RouteNote),
	}
}

func (s *GRPCServer) GetFeature(ctx context.Context, point *api.Point) (*api.Feature, error) {
	domainPoint := model.Point{Latitude: point.Latitude, Longitude: point.Longitude}
	feature, err := s.featureSvc.GetFeature(ctx, domainPoint)
	if err != nil {
		return nil, err
	}

	featureProto := &api.Feature{
		Location: &api.Point{
			Latitude:  domainPoint.Latitude,
			Longitude: domainPoint.Longitude,
		},
		Name: feature.Name,
	}

	return featureProto, nil
}

func (s *GRPCServer) ListFeatures(rect *api.Rectangle, stream api.RouteGuide_ListFeaturesServer) error {
	domainRect := model.Rectangle{
		Lo: model.Point{Latitude: rect.Lo.Latitude, Longitude: rect.Lo.Longitude},
		Hi: model.Point{Latitude: rect.Hi.Latitude, Longitude: rect.Hi.Longitude},
	}
	features, err := s.featureSvc.ListFeatures(stream.Context(), domainRect)
	if err != nil {
		return err
	}

	for _, feature := range features {
		featureProto := &api.Feature{
			Location: &api.Point{
				Latitude:  feature.Location.Latitude,
				Longitude: feature.Location.Latitude,
			},
			Name: feature.Name,
		}

		if err := stream.Send(featureProto); err != nil {
			return err
		}
	}
	return nil
}

func (s *GRPCServer) RecordRoute(stream api.RouteGuide_RecordRouteServer) error {
	var points []model.Point
	for {
		point, err := stream.Recv()
		if err == io.EOF {
			summary, err := s.featureSvc.RecordRoute(stream.Context(), points)
			if err != nil {
				return err
			}
			return stream.SendAndClose(&api.RouteSummary{
				PointCount:   summary.PointCount,
				FeatureCount: summary.FeatureCount,
				Distance:     summary.Distance,
				ElapsedTime:  summary.ElapsedTime,
			})
		}
		if err != nil {
			return err
		}
		points = append(points, model.Point{Latitude: point.Latitude, Longitude: point.Longitude})
	}
}

func (s *GRPCServer) RouteChat(stream api.RouteGuide_RouteChatServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		key := serialize(in.Location)

		s.mu.Lock()
		s.routeNotes[key] = append(s.routeNotes[key], in)
		// Note: this copy prevents blocking other clients while serving this one.
		// We don't need to do a deep copy, because elements in the slice are
		// insert-only and never modified.
		rn := make([]*api.RouteNote, len(s.routeNotes[key]))
		copy(rn, s.routeNotes[key])
		s.mu.Unlock()

		for _, note := range rn {
			if err := stream.Send(note); err != nil {
				return err
			}
		}
	}
}

func serialize(point *api.Point) string {
	return fmt.Sprintf("%d %d", point.Latitude, point.Longitude)
}
