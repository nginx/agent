/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package main

import (
	"github.com/nginx/agent/v3/internal"
)

func main() {
	app := internal.NewApp()
	app.Run()
}
