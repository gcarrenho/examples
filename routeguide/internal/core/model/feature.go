package model

type Feature struct {
	Location Point
	Name     string
}

type Point struct {
	Latitude  int32
	Longitude int32
}

type Rectangle struct {
	Lo Point
	Hi Point
}

/*
func ProtoToDomain(protoLoc *pb.Location) *domain.Location {
    return &domain.Location{
        Latitude:  protoLoc.Latitude,
        Longitude: protoLoc.Longitude,
    }
}

func DomainToProto(domainLoc *domain.Location) *pb.Location {
    return &pb.Location{
        Latitude:  domainLoc.Latitude,
        Longitude: domainLoc.Longitude,
    }
}*/
