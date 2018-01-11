package main

import (
	"github.com/mwitkow/grpc-proxy/proxy"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"github.com/golang/glog"
	"strings"
)

func NewGRPCProxyServer(upstream string) *grpc.Server {
	director := func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		// fullMethodName would be like "/yourpackage.YourService/YourMethod"
		// TODO map fullMethodName to `nonResourceURL`
		// TODO assume the `verb` to be "get" anyway
		glog.Infof("Method: %s", fullMethodName)
		md, ok := metadata.FromIncomingContext(ctx)
		glog.Infof("Metadata: %v", md)

		// Copy the inbound metadata explicitly.
		outCtx, _ := context.WithCancel(ctx)
		outCtx = metadata.NewOutgoingContext(outCtx, md.Copy())

		authz, exists := md["authorization"]
		if !exists {
			return outCtx, nil, grpc.Errorf(codes.Unauthenticated, "Missing authorization header")
		}

		if len(authz) != 1 {
			return outCtx, nil, grpc.Errorf(codes.InvalidArgument, "Too many values for authorization header")
		}

		splits := strings.Split(authz[0], " ")
		if len(splits) != 2 {
			return outCtx, nil, grpc.Errorf(codes.Unauthenticated, "Missing bearer token for authentication")
		}
		token := splits[1]

		// TODO Do this only when in DEBUG mode
		glog.Infof("Authenticating with the token: %s", token)

		// TODO Authenticate with the bearer token included in the `authorization` metadata
		// authz.AuthenticateToken(token)

		// TODO Authorize it via a token review
		// authz.Authorize(metadataToAttributes(md))

		// Make sure we use DialContext so the dialing can be cancelled/time out together with the context.
		// TODO Pool connections for efficiency
		conn, err := grpc.DialContext(ctx, upstream, grpc.WithCodec(proxy.Codec()))
		return outCtx, conn, err

		// Note: We have to return the first arg=outCtx like this anyway.
		// If it was nil, `mwitkow/grpc-proxy/proxy/handler.go:70` panics
		return outCtx, nil, grpc.Errorf(codes.Unimplemented, "Unknown method")
	}

	return grpc.NewServer(
		grpc.CustomCodec(proxy.Codec()),
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)))
}
