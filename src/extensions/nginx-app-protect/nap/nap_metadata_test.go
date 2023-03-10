/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
)

const (
	configFile   = "/tmp/testdata/nginx.conf"
	basePath     = "/tmp/testdata"
	metadataPath = "/tmp/testdata/nms"
	metadataFile = "/tmp/testdata/nms/app_protect_metadata.json"

	nginxID  = "1"
	systemID = "2"

	wafVersion                 = "4.2.0"
	wafAttackSignaturesVersion = "2023.01.01"
	wafThreatCampaignsVersion  = "2023.01.02"
)

var (
	config = `daemon  off;
	worker_processes  2;
	user              www-data;

	http {
		access_log    /tmp/testdata/logs/access1.log  $upstream_time;
		app_protect_enable on;
		app_protect_security_log_enable on;

		server {
			server_name   localhost;
			listen        127.0.0.1:80;

			location      / {
				root      /tmp/testdata/root;
				app_protect_policy_file /tmp/testdata/root/my-nap-policy.json;
				app_protect_security_log /tmp/testdata/root/log-all.json /var/log/security.log;
			}
		}

		access_log    /tmp/testdata/logs/access2.log  combined;
	}`

	metadata1 = `{
	"napVersion": "4.2.0",
	"precompiledPublication": false,
	"attackSignatureRevisionTimestamp": "2023.01.09",
	"threatCampaignRevisionTimestamp": "2023.01.04",
	"policyMetadata": [
		{
			"name": "NginxStrictPolicy.tgz"
		},
		{
			"name": "NginxDefaultPolicy.tgz"
		}
	],
	"logProfileMetadata": [
		{
			"name": "log_blocked.tgz"
		}
	]
}`

	metadata2 = `{
	"napVersion":"4.2.0",
	"precompiledPublication": true,
	"attackSignatureRevisionTimestamp": "2023.01.09",
	"threatCampaignRevisionTimestamp": "2023.01.04",
	"policyMetadata": [
		{
			"name": "NginxStrictPolicy.tgz"
		},
		{
			"name": "NginxDefaultPolicy.tgz"
		}
	],
	"logProfileMetadata": [
		{
			"name": "log_blocked.tgz"
		}
	]
}`

	expectedFalse = `{"napVersion":"4.2.0","precompiledPublication":false,"attackSignatureRevisionTimestamp":"2023.01.01","threatCampaignRevisionTimestamp":"2023.01.02","policyMetadata":[{"name":"my-nap-policy.json"}],"logProfileMetadata":[{"name":"log-all.json"}]}`

	expectedTrue = `{"napVersion":"4.2.0","precompiledPublication":true,"attackSignatureRevisionTimestamp":"2023.01.01","threatCampaignRevisionTimestamp":"2023.01.02","policyMetadata":[{"name":"my-nap-policy.json"}],"logProfileMetadata":[{"name":"log-all.json"}]}`
)

func TestUpdateNapMetadata(t *testing.T) {
	testCases := []struct {
		testName   string
		meta       string
		precompPub bool
		expected   string
	}{
		{
			testName:   "NoMetadataDir",
			meta:       "",
			precompPub: false,
			expected:   expectedFalse,
		},
		{
			testName:   "NoMetadataFile",
			meta:       "",
			precompPub: false,
			expected:   expectedFalse,
		},
		{
			testName:   "NoMetadataFileChange",
			meta:       metadata2,
			precompPub: true,
			expected:   metadata2,
		},
		{
			testName:   "PrecompilationWasFalse",
			meta:       metadata1,
			precompPub: true,
			expected:   expectedTrue,
		},
		{
			testName:   "PrecompilationWasTrue",
			meta:       metadata2,
			precompPub: false,
			expected:   expectedFalse,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			defer tearDownTestDirectory()

			switch tc.testName {
			case "NoMetadataDir":
				err := os.MkdirAll(basePath, 0755)
				assert.NoError(t, err)
			case "NoMetadataFile":
				err := os.MkdirAll(metadataPath, 0755)
				assert.NoError(t, err)
			default:
				err := setUpFile(metadataFile, []byte(tc.meta))
				assert.NoError(t, err)
			}

			err := setUpFile(configFile, []byte(config))
			assert.NoError(t, err)
			allowedDirs := map[string]struct{}{}

			cfg, err := sdk.GetNginxConfig(configFile, nginxID, systemID, allowedDirs)
			assert.NoError(t, err)

			appProtectWAFDetails := &proto.AppProtectWAFDetails{
				WafVersion:              wafVersion,
				AttackSignaturesVersion: wafAttackSignaturesVersion,
				ThreatCampaignsVersion:  wafThreatCampaignsVersion,
				WafLocation:             metadataFile,
				PrecompiledPublication:  tc.precompPub,
			}

			err = UpdateMetadata(cfg, appProtectWAFDetails)
			assert.NoError(t, err)

			data, err := os.ReadFile(metadataFile)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, string(data))
		})
	}
}

func setUpFile(file string, content []byte) error {
	err := os.MkdirAll(filepath.Dir(file), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func tearDownTestDirectory() {
	os.RemoveAll("/tmp/testdata")
}
