/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

type Plugin interface {
	Init(MessagePipeInterface)
	Close()
	Process(*Message)
	Info() *Info
	Subscriptions() []string
}

type ExtensionPlugin = Plugin
