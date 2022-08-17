package client

import (
	"context"
	"fmt"
)

func NewClientController() Controller {
	return &ctrl{}
}

type ctrl struct {
	ctx     context.Context
	clients []Client
}

func (c *ctrl) WithClient(client Client) Controller {
	c.clients = append(c.clients, client)

	return c
}

func (c *ctrl) WithContext(ctx context.Context) Controller {
	c.ctx = ctx

	return c
}

func (c *ctrl) Connect() error {
	var retErr error
	for _, client := range c.clients {
		if err := client.Connect(c.ctx); err != nil {
			if retErr == nil {
				retErr = fmt.Errorf("%s failed to connect: %w", client.Server(), err)
			} else {
				retErr = fmt.Errorf("%v\n%s failed to connect: %w", retErr, client.Server(), err)
			}
		}
	}

	return retErr
}

func (c *ctrl) Close() error {
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
