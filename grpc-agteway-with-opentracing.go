package grpc_gateway

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

func injectHeadersIntoMetadata(ctx context.Context, req *http.Request) metadata.MD {
	var (
		otHeaders = []string{
			"x-request-id",
			"x-b3-traceid",
			"x-b3-spanid",
			"x-b3-parentspanid",
			"x-b3-sampled",
			"x-b3-flags",
			"x-ot-span-context"}
	)
	var pairs []string

	for _, h := range otHeaders {
		if v := req.Header.Get(h); len(v) > 0 {
			pairs = append(pairs, h, v)
		}
	}
	return metadata.Pairs(pairs...)
}

type annotator func(context.Context, *http.Request) metadata.MD

func chainGrpcAnnotators(annotators ...annotator) annotator {
	return func(c context.Context, r *http.Request) metadata.MD {
		var mds []metadata.MD
		for _, a := range annotators {
			mds = append(mds, a(c, r))
		}
		return metadata.Join(mds...)
	}
}

type GatewayConfig struct {
	Fn       func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error)
	GrpcPort string
	HttpPort string
	Mux      *runtime.ServeMux
	Opts     []grpc.DialOption
}

func RegisterGRPCGateway(cf *GatewayConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	annotators := []annotator{injectHeadersIntoMetadata}

	if cf.GrpcPort == "" {
		return errors.New("grpc port invalid")
	}

	if cf.HttpPort == "" {
		cf.HttpPort = ":8088"
	}

	if cf.Mux == nil {
		cf.Mux = runtime.NewServeMux(
			runtime.WithMetadata(chainGrpcAnnotators(annotators...)),
		)
	}

	if len(cf.Opts) == 0 {
		cf.Opts = DialOptionGRPC()
	}

	err := cf.Fn(ctx, cf.Mux, cf.GrpcPort, cf.Opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(cf.HttpPort, cf.Mux)
}

func DialOptionGRPC() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    time.Minute,
			Timeout: time.Second * 30,
		}),
	}
}
