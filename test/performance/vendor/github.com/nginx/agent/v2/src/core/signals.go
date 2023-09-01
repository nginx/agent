/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nginx/agent/sdk/v2/agent/events"
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/v2/src/core/config"
	log "github.com/sirupsen/logrus"
)

// handleSignals handles signals to attempt graceful shutdown
// for now it also handles sending the agent stopped event because as of today we don't have a mechanism for synchronizing
// tasks between multiple plugins from outside a plugin
func HandleSignals(
	ctx context.Context,
	cmder client.Commander,
	loadedConfig *config.Config,
	env Environment,
	pipe MessagePipeInterface,
	cancel context.CancelFunc,
	controller client.Controller,
) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			event := events.NewAgentEventMeta(config.MODULE,
				version,
				strconv.Itoa(os.Getpid()),
				"Initialize Agent",
				env.GetHostname(),
				env.GetSystemUUID(),
				loadedConfig.InstanceGroup,
				loadedConfig.Tags)

			stopCmd := events.GenerateAgentStopEventCommand(event)
			log.Debugf("Sending agent stopped event: %v", stopCmd)

			if cmder == nil {
				log.Warn("Command channel not configured. Skipping sending AgentStopped event")
			} else if err := cmder.Send(ctx, client.MessageFromCommand(stopCmd)); err != nil {
				log.Errorf("Error sending AgentStopped event to command channel: %v", err)
			}

			if controller != nil {
				if err := controller.Close(); err != nil {
					log.Warnf("Unable to close controller: %v", err)
				}
			}

			log.Warn("NGINX Agent exiting")
			cancel()

			timeout := time.Second * 5
			time.Sleep(timeout)
			log.Fatalf("Failed to gracefully shutdown within timeout of %v. Exiting", timeout)
		case <-ctx.Done():
		}
	}()
}
