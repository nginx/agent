package sdk

import (
	"github.com/nginxinc/nginx-go-crossplane"
)

type CrossplaneTraverseCallback = func(parent *crossplane.Directive, current *crossplane.Directive) bool
type CrossplaneTraverseCallbackStr = func(parent *crossplane.Directive, current *crossplane.Directive) string

func traverse(root *crossplane.Directive, callback CrossplaneTraverseCallback, stop *bool) {
	if *stop {
		return
	}
	for _, child := range root.Block {
		if !callback(root, child) {
			*stop = true
			return
		}
		traverse(child, callback, stop)
		if *stop {
			return
		}
	}
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

func CrossplaneConfigTraverse(root *crossplane.Config, callback CrossplaneTraverseCallback) {
	stop := false
	for _, dir := range root.Parsed {
		if !callback(nil, dir) {
			return
		}
		traverse(dir, callback, &stop)
	}
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
