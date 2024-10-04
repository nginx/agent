/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sdk

import (
	"github.com/nginxinc/nginx-go-crossplane"
)

type (
	CrossplaneTraverseCallback    = func(parent *crossplane.Directive, current *crossplane.Directive) (bool, error)
	CrossplaneTraverseCallbackStr = func(parent *crossplane.Directive, current *crossplane.Directive) string
)

func traverse(root *crossplane.Directive, callback CrossplaneTraverseCallback, stop *bool) error {
	if *stop {
		return nil
	}
	for _, child := range root.Block {
		result, err := callback(root, child)
		if err != nil {
			return err
		}

		if !result {
			*stop = true
			return nil
		}

		err = traverse(child, callback, stop)
		if err != nil {
			return err
		}

		if *stop {
			return nil
		}
	}
	return nil
}

func traverseStr(root *crossplane.Directive, callback CrossplaneTraverseCallbackStr, stop *bool) string {
	response := ""
	if *stop {
		return ""
	}
	for _, child := range root.Block {
		response = callback(root, child)
		if response != "" {
			*stop = true
			return response
		}
		response = traverseStr(child, callback, stop)
		if *stop {
			return response
		}
	}
	return response
}

func CrossplaneConfigTraverse(root *crossplane.Config, callback CrossplaneTraverseCallback) error {
	stop := false
	for _, dir := range root.Parsed {
		result, err := callback(nil, dir)
		if err != nil {
			return err
		}

		if !result {
			return nil
		}

		err = traverse(dir, callback, &stop)
		if err != nil {
			return err
		}
	}
	return nil
}

func CrossplaneConfigTraverseStr(root *crossplane.Config, callback CrossplaneTraverseCallbackStr) string {
	stop := false
	response := ""
	for _, dir := range root.Parsed {
		response = callback(nil, dir)
		if response != "" {
			return response
		}
		response = traverseStr(dir, callback, &stop)
		if response != "" {
			return response
		}
	}
	return response
}
