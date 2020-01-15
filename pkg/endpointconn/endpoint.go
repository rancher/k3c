package endpointconn

import (
	"context"
	"net"
	"net/url"
	"strings"

	"google.golang.org/grpc"
)

func Get(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	if !strings.HasPrefix(endpoint, "unix://") {
		endpoint = "unix://" + endpoint
	}
	target, dialer, err := targetAndDialer(endpoint)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.DialContext(ctx, target, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithContextDialer(dialer), grpc.FailOnNonTempDialError(true))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func targetAndDialer(endpoint string) (string, func(context.Context, string) (net.Conn, error), error) {
	network, address, err := parseEndpoint(endpoint)
	if err != nil {
		return "", nil, err
	}

	return address, func(ctx context.Context, address string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, address)
	}, nil
}

func parseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "unix", endpoint, nil
	}

	switch u.Scheme {
	case "unix":
		return "unix", u.Path, nil
	default:
		return u.Scheme, u.Host, nil
	}
}
