package dgraph

import (
	"context"
	"net"
	"strconv"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/log"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

// Connect establishes a connection with the dgraph client.
func Connect(ctx context.Context, config config.Dgraph) (*dgo.Dgraph, func() error, error) {
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	opts := []grpc.DialOption{
		// TODO: Set transport security
		// grpc.WithTransportCredentials(credentials.NewServerTLSFromCert(&config.TLSCertificates[0])),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
	}

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "connecting to Dgraph Alpha")
	}

	dc := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	if err := dc.Alter(ctx, &api.Operation{Schema: schema}); err != nil {
		return nil, nil, errors.Wrap(err, "creating schema")
	}

	log.Sugar().Infof("Connected to dgraph on %s", addr)
	return dc, conn.Close, nil
}

const schema = `
type Event {
	event_id
	liked_by
	invited
	confirmed
	banned
}

type User {
	user_id
	following
	blocked
}

event_id: string @index(hash) .
liked_by: [uid] @reverse .
invited: [uid] @reverse .
confirmed: [uid] @reverse .
banned: [uid] @reverse .

user_id: string @index(hash) .
following: [uid] @reverse .
blocked: [uid] @reverse .`
