/*
* Copyright (C) F5 Inc. 2022
* All rights reserved.
*
* No part of the software may be reproduced or transmitted in any
* form or by any means, electronic or mechanical, for any purpose,
* without express written permission of F5 Inc.
 */

package processor

import (
	"sync"
)

// sigMap is a mapping structure for signatureIDs to signatureNames
type sigMap struct {
	mux sync.RWMutex
	m   map[string]string
}

func (s *sigMap) Get(id string) (string, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	val, ok := s.m[id]
	return val, ok
}

func (s *sigMap) Update(newMap map[string]string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.m = newMap
}
