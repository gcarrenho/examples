package adapters

import (
	"context"
	"fmt"
	"io"
	"sync"

	api "github.com/gcarrenho/routeguide/api/v1"
	"github.com/gcarrenho/routeguide/internal/core/model"
	"github.com/gcarrenho/routeguide/internal/core/ports"
	"google.golang.org/grpc"
)

var _ api.RouteGuideServer = (*GRPCServer)(nil)

type Config struct {
	FeatureSvc ports.FeatureService
	mu         sync.Mutex // protects routeNotes
	RouteNotes map[string][]*api.RouteNote
}

type GRPCServer struct {
	api.UnimplementedRouteGuideServer
	*Config
}

func newGRPCServer(config *Config) (srv *GRPCServer, err error) {
	srv = &GRPCServer{
		Config: config,
	}
	srv.Config.RouteNotes = make(map[string][]*api.RouteNote)
	return srv, nil
}

func NewGRPCServer(config *Config, grpcOpts ...grpc.ServerOption) (
	*grpc.Server,
	error,
) {
	gsrv := grpc.NewServer(grpcOpts...) // Create the server gRPC with grpcOpts

	srv, err := newGRPCServer(config) // Create an instance of gRPC server with config
	if err != nil {
		return nil, err
	}
	api.RegisterRouteGuideServer(gsrv, srv) // Registers  the our server(srv) in the gRPC server under th "LogServer" services defined in api.

	return gsrv, nil
}

func (s *GRPCServer) GetFeature(ctx context.Context, point *api.Point) (*api.Feature, error) {
	domainPoint := model.Point{Latitude: point.Latitude, Longitude: point.Longitude}
	feature, err := s.Config.FeatureSvc.GetFeature(ctx, domainPoint)
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
	features, err := s.Config.FeatureSvc.ListFeatures(stream.Context(), domainRect)
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
			summary, err := s.Config.FeatureSvc.RecordRoute(stream.Context(), points)
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
		//Maybe we need a service method to do this.

		key := serialize(in.Location)

		s.mu.Lock()
		s.Config.RouteNotes[key] = append(s.Config.RouteNotes[key], in)
		// Note: this copy prevents blocking other clients while serving this one.
		// We don't need to do a deep copy, because elements in the slice are
		// insert-only and never modified.
		rn := make([]*api.RouteNote, len(s.Config.RouteNotes[key]))
		copy(rn, s.Config.RouteNotes[key])
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
