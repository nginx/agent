/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
)

func NewClientController() Controller {
	return &ctrl{}
}

type ctrl struct {
	ctx     context.Context
	cncl    context.CancelFunc
	clients []Client
}

func (c *ctrl) WithClient(client Client) Controller {
	c.clients = append(c.clients, client)

	return c
}

func (c *ctrl) WithContext(ctx context.Context) Controller {
	c.ctx, c.cncl = context.WithCancel(ctx)
	return c
}

func (c *ctrl) Connect() {
	for _, client := range c.clients {
		if err := client.Connect(c.ctx); err != nil {
			log.Warnf("%s failed to connect: %v", client.Server(), err)
		}
	}
}

func (c *ctrl) Close() error {
	defer c.cncl()
	var retErr error
	for _, client := range c.clients {
		if err := client.Close(); err != nil {
			if retErr == nil {
				retErr = fmt.Errorf("%s failed to close: %w", client.Server(), err)
			} else {
				retErr = fmt.Errorf("%v\n%s failed to close: %w", retErr, client.Server(), err)
			}
		}
	}

	return retErr
}

func (c *ctrl) Context() context.Context {
	return c.ctx
}
