/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package main

import (
	"os"

	"github.com/nginx/agent/v3/internal"
)

func main() {
	app := internal.NewApp()
	err := app.Run()
	if err != nil {
		os.Exit(1)
	}
}
