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
	if err := CreateSchema(ctx, dc); err != nil {
		return nil, nil, err
	}

	log.Sugar().Infof("Connected to dgraph on %s", addr)
	return dc, conn.Close, nil
}

// CreateSchema creates the dgraph schema.
func CreateSchema(ctx context.Context, dc *dgo.Dgraph) error {
	if err := dc.Alter(ctx, &api.Operation{Schema: schema}); err != nil {
		return errors.Wrap(err, "creating schema")
	}

	return nil
}

const schema = `
type Event {
	event_id
	liked_by
	invited
	banned
}

type User {
	user_id
	friend
	follows
	blocked
}

type Post {
	post_id
	liked_by
}

type Comment {
	comment_id
	liked_by
}

liked_by: [uid] @reverse .

event_id: string @index(hash) .
invited: [uid] @reverse .
banned: [uid] @reverse .

post_id: string @index(hash) .
comment_id: string @index(hash) .

user_id: string @index(hash) .
friend: [uid] .
follows: [uid] .
blocked: [uid] @reverse .`
