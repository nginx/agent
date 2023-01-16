/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

import (
	"path"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"

	"github.com/nginxinc/nginx-go-crossplane"
)

// getContent parses the config for NAP policies and profiles
func getContent(cfg *proto.NginxConfig) ([]string, []string) {
	policyMap := make(map[string]bool)
	profileMap := make(map[string]bool)

	for _, directory := range cfg.GetDirectoryMap().GetDirectories() {
		for _, file := range directory.GetFiles() {
			confFile := path.Join(directory.GetName(), file.GetName())
			payload, err := crossplane.Parse(confFile,
				&crossplane.ParseOptions{
					SingleFile:         false,
					StopParsingOnError: true,
				},
			)
			if err != nil {
				continue
			}
			for _, conf := range payload.Config {
				err = sdk.CrossplaneConfigTraverse(&conf,
					func(parent *crossplane.Directive, directive *crossplane.Directive) (bool, error) {
						switch directive.Directive {
						case "app_protect_policy_file":
							if len(directive.Args) == 1 {
								_, policy := path.Split(directive.Args[0])
								policyMap[policy] = true
							}
						case "app_protect_security_log":
							if len(directive.Args) == 2 {
								_, profile := path.Split(directive.Args[0])
								profileMap[profile] = true
							}
						}
						return true, nil
					})
				if err != nil {
					continue
				}
			}
			if err != nil {
				continue
			}
		}
	}
	policies := []string{}
	for policy, _ := range policyMap {
		policies = append(policies, policy)
	}
	profiles := []string{}
	for profile, _ := range profileMap {
		profiles = append(profiles, profile)
	}

	return policies, profiles
}
