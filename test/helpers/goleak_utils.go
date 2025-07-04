// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import "go.uber.org/goleak"

var GoLeakOptions = []goleak.Option{
	goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
}
