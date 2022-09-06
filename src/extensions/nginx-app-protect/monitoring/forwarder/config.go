/*
 * Copyright (C) F5 Inc. 2022
 * All rights reserved.
 *
 * No part of the software may be reproduced or transmitted in any
 * form or by any means, electronic or mechanical, for any purpose,
 * without express written permission of F5 Inc.
 */

package forwarder

import "github.com/sirupsen/logrus"

// Config holds the config for Forwarder.
type Config struct {
	Logger *logrus.Entry
}
