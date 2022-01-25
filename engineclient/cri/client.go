// Copyright 2022 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cri

import (
	"context"
	"time"

	criapis "k8s.io/cri-api/pkg/apis"

	"github.com/containerd/containerd/pkg/dialer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

// Client is a CRI runtime service API client. Unfortunately, there isn't a
// generally reusable CRI client available, despite crictl and k8s itself, so we
// need to roll our own.
type Client struct {
	criapis.RuntimeService

	conn *grpc.ClientConn
}

type clientOpts struct {
	timeout     time.Duration
	dialOptions []grpc.DialOption
}

type ClientOpt func(c *clientOpts) error

// WithTimeout sets the connection timeout for the client.
func WithTimeout(d time.Duration) ClientOpt {
	return func(c *clientOpts) error {
		c.timeout = d
		return nil
	}
}

// WithDialOpts allows grpc.DialOptions to be set on the client connection.
func WithDialOpts(opts []grpc.DialOption) ClientOpt {
	return func(c *clientOpts) error {
		c.dialOptions = opts
		return nil
	}
}

// New returns a new CRI API client that is connected to the CRI service
// instance provided by address.
func New(address string, opts ...ClientOpt) (*Client, error) {
	var clopts clientOpts
	for _, opt := range opts {
		if err := opt(&clopts); err != nil {
			return nil, err
		}
	}
	if clopts.timeout == 0 {
		clopts.timeout = 10 * time.Second
	}

	cl := &Client{}

	if address != "" {
		backoffConfig := backoff.DefaultConfig
		backoffConfig.MaxDelay = 3 * time.Second
		connParams := grpc.ConnectParams{
			Backoff: backoffConfig,
		}
		gopts := []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithInsecure(),
			grpc.FailOnNonTempDialError(true),
			grpc.WithConnectParams(connParams),
			grpc.WithContextDialer(dialer.ContextDialer),
		}
		if len(clopts.dialOptions) > 0 {
			gopts = clopts.dialOptions
		}
		connector := func() (*grpc.ClientConn, error) {
			ctx, cancel := context.WithTimeout(context.Background(), clopts.timeout)
			defer cancel()
			conn, err := grpc.DialContext(ctx, dialer.DialAddress(address), gopts...)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to dial %q", address)
			}
			return conn, nil
		}
		conn, err := connector()
		if err != nil {
			return nil, err
		}
		cl.conn = conn
	}

	return cl, nil
}
