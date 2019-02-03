package tbdClient

import (
	"context"
	"crypto/tls"

	"github.com/childoftheuniverse/fancylocking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

/*
TokenBucketClient is a client for a networked token bucket implementation.
*/
type TokenBucketClient struct {
	mtx    fancylocking.MutexWithDeadline
	conn   *grpc.ClientConn
	client TokenBucketServiceClient
}

/*
NewTokenBucketClient creates a new TokenBucketClient for the specified
remote address. If a tlsConfig is passed, then TLS is enabled, otherwise
the client will be run in insecure mode.
*/
func NewTokenBucketClient(
	remoteAddr string, tlsConfig *tls.Config, opts ...grpc.DialOption) (
	*TokenBucketClient, error) {
	var conn *grpc.ClientConn
	var err error

	if tlsConfig == nil {
		opts = append(opts, grpc.WithInsecure())
	} else {
		opts = append(opts, grpc.WithTransportCredentials(
			credentials.NewTLS(tlsConfig)))
	}

	if conn, err = grpc.Dial(remoteAddr, opts...); err != nil {
		return nil, err
	}

	return &TokenBucketClient{
		mtx:    fancylocking.NewMutexWithDeadline(),
		conn:   conn,
		client: NewTokenBucketServiceClient(conn),
	}, nil
}

/*
MultiTokenRequest sends a MultiTokenBucketRequest to the server and
just returns the response.
*/
func (tbc *TokenBucketClient) MultiTokenRequest(
	ctx context.Context, in *MultiTokenBucketRequest, opts ...grpc.CallOption) (
	*MultiTokenBucketResponse, error) {
	if !tbc.mtx.LockWithContext(ctx) {
		return nil, ctx.Err()
	}
	return tbc.client.MultiTokenRequest(ctx, in, opts...)
}

/*
TokenRequest creates a MultiTokenBucketRequest for the parameters passed in
and sends it to the server; the result of whether the request passed or failed
is being returned as a boolean. In any case, the result returned fails open.
*/
func (tbc *TokenBucketClient) TokenRequest(
	ctx context.Context, family, bucket string, amount int64) (bool, error) {
	var mresp *MultiTokenBucketResponse
	var resp *TokenBucketResponse
	var opts = []grpc.CallOption{
		grpc.FailFast(true),
	}
	var req = &TokenBucketRequest{
		BucketFamily:       family,
		Bucket:             bucket,
		Amount:             amount,
		PartialFulfillment: false,
	}
	var mreq = &MultiTokenBucketRequest{
		Request:    []*TokenBucketRequest{req},
		RequireAll: false,
	}
  var err error

	if mresp, err = tbc.MultiTokenRequest(ctx, mreq, opts...); err != nil {
		return true, err
	}

	for _, resp = range mresp.Response {
		return resp.Success, nil
	}

	/* Fail open in case the response was empty for some reason. */
	return true, nil
}
