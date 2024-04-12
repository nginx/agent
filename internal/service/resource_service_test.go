// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"strings"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
)

const fedoraOsReleaseInfo = `
 NAME=Fedora
 VERSION="32 (Workstation Edition)"
 ID=fedora
 VERSION_ID=32
 PRETTY_NAME="Fedora 32 (Workstation Edition)"
 ANSI_COLOR="0;38;2;60;110;180"
 LOGO=fedora-logo-icon
 CPE_NAME="cpe:/o:fedoraproject:fedora:32"
 HOME_URL="https://fedoraproject.org/"
 DOCUMENTATION_URL="https://docs.fedoraproject.org/en-US/fedora/f32/system-administrators-guide/"
 SUPPORT_URL="https://fedoraproject.org/wiki/Communicating_and_getting_help"
 BUG_REPORT_URL="https://bugzilla.redhat.com/"
 REDHAT_BUGZILLA_PRODUCT="Fedora"
 REDHAT_BUGZILLA_PRODUCT_VERSION=32
 REDHAT_SUPPORT_PRODUCT="Fedora"
 REDHAT_SUPPORT_PRODUCT_VERSION=32
 PRIVACY_POLICY_URL="https://fedoraproject.org/wiki/Legal:PrivacyPolicy"
 VARIANT="Workstation Edition"
 VARIANT_ID=workstation"
 `

const ubuntuReleaseInfo = `
 NAME="Ubuntu"
 VERSION="20.04.5 LTS (Focal Fossa)"
 VERSION_ID="20.04"
 ID=ubuntu
 ID_LIKE=debian
 PRETTY_NAME="Ubuntu 20.04.5 LTS"
 HOME_URL="https://www.ubuntu.com"
 SUPPORT_URL=\"https://help.ubuntu.com/"
 BUG_REPORT_URL=\"https://bugs.launchpad.net/ubuntu/"
 PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
 VERSION_CODENAME=focal
 UBUNTU_CODENAME=focal
 `

const osReleaseInfoWithNoName = `
 VERSION="20.04.5 LTS (Focal Fossa)"
 VERSION_ID="20.04"
 ID=ubuntu
 ID_LIKE=debian
 PRETTY_NAME="Ubuntu 20.04.5 LTS"
 HOME_URL="https://www.ubuntu.com"
 SUPPORT_URL=\"https://help.ubuntu.com/"
 BUG_REPORT_URL=\"https://bugs.launchpad.net/ubuntu/"
 PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
 VERSION_CODENAME=focal
 UBUNTU_CODENAME=focal
 `

func TestResourceService_GetResource(t *testing.T) {
	// ctx := context.Background()

	// containerInfo := protos.GetContainerizedResource()
	// hostInfo := protos.GetHostResource()

	// mockInfo := &hostfakes.FakeInfoInterface{}
	// mockInfo.GetContainerInfoReturns(containerInfo.Info)
	// mockInfo.GetHostInfoReturns(hostInfo)

	// resourceService := NewResourceService()
	// resourceService.info = mockInfo

	// // Test Container
	// mockInfo.IsContainerReturns(true)

	// resource := resourceService.GetResource(ctx)

	// assert.Equal(t, &v1.Resource{Id: "123", Info: containerInfo}, resource)

	// // Test VM
	// mockInfo.IsContainerReturns(false)

	// resource = resourceService.GetResource(ctx)

	// assert.Equal(t, &v1.Resource{Id: "123", Info: hostInfo}, resource)
}

func TestParseOsReleaseFile(t *testing.T) {
	tests := []struct {
		name             string
		osReleaseContent string
		expect           map[string]string
	}{
		{
			name:             "ubuntu os-release info",
			osReleaseContent: ubuntuReleaseInfo,
			expect: map[string]string{
				"VERSION_ID":       "20.04",
				"VERSION":          "20.04.5 LTS (Focal Fossa)",
				"VERSION_CODENAME": "focal",
				"NAME":             "Ubuntu",
				"ID":               "ubuntu",
			},
		},
		{
			name:             "fedora os-release info",
			osReleaseContent: fedoraOsReleaseInfo,
			expect: map[string]string{
				"VERSION_ID": "32",
				"VERSION":    "32 (Workstation Edition)",
				"NAME":       "Fedora",
				"ID":         "fedora",
			},
		},
		{
			name:             "os-release info with no name",
			osReleaseContent: osReleaseInfoWithNoName,
			expect: map[string]string{
				"VERSION_ID":       "20.04",
				"VERSION":          "20.04.5 LTS (Focal Fossa)",
				"VERSION_CODENAME": "focal",
				"NAME":             "unix",
				"ID":               "ubuntu",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceService := NewResourceService()

			reader := strings.NewReader(tt.osReleaseContent)
			osRelease, _ := resourceService.parseOsReleaseFile(reader)
			for releaseInfokey := range tt.expect {
				assert.Equal(t, osRelease[releaseInfokey], tt.expect[releaseInfokey])
			}
		})
	}
}

func TestMergeHostAndOsReleaseInfo(t *testing.T) {
	tests := []struct {
		name        string
		hostRelease *v1.ReleaseInfo
		osRelease   map[string]string
		expect      *v1.ReleaseInfo
	}{
		{
			name: "os-release info present",
			hostRelease: &v1.ReleaseInfo{
				VersionId: "20.04",
				Version:   "5.15.0-1028-aws",
				Codename:  "linux",
				Name:      "debian",
				Id:        "ubuntu",
			},
			osRelease: map[string]string{
				"VERSION_ID":       "20.04",
				"VERSION":          "20.04.5 LTS (Focal Fossa)",
				"VERSION_CODENAME": "focal",
				"NAME":             "Ubuntu",
				"ID":               "ubuntu",
			},
			expect: &v1.ReleaseInfo{
				VersionId: "20.04",
				Version:   "20.04.5 LTS (Focal Fossa)",
				Codename:  "focal",
				Name:      "Ubuntu",
				Id:        "ubuntu",
			},
		},
		{
			name: "os-release info value missing",
			osRelease: map[string]string{
				"VERSION_ID":       "32",
				"VERSION":          "32 (Workstation Edition)",
				"VERSION_CODENAME": "",
				"NAME":             "Fedora",
				"ID":               "fedora",
			},
			hostRelease: &v1.ReleaseInfo{
				VersionId: "32",
				Version:   "Fedora 32 (Workstation Edition)",
				Codename:  "fedora",
				Name:      "Fedora",
				Id:        "Fedora",
			},
			expect: &v1.ReleaseInfo{
				VersionId: "32",
				Version:   "32 (Workstation Edition)",
				Codename:  "fedora",
				Name:      "Fedora",
				Id:        "fedora",
			},
		},
		{
			name: "os-release info field missing",
			osRelease: map[string]string{
				"VERSION_ID": "32",
				"VERSION":    "32 (Workstation Edition)",
				"NAME":       "Fedora",
				"ID":         "fedora",
			},
			hostRelease: &v1.ReleaseInfo{
				VersionId: "32",
				Version:   "Fedora 32 (Workstation Edition)",
				Codename:  "fedora",
				Name:      "Fedora",
				Id:        "Fedora",
			},
			expect: &v1.ReleaseInfo{
				VersionId: "32",
				Version:   "32 (Workstation Edition)",
				Codename:  "fedora",
				Name:      "Fedora",
				Id:        "fedora",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceService := NewResourceService()

			releaseInfo := resourceService.mergeHostAndOsReleaseInfo(tt.hostRelease, tt.osRelease)
			assert.Equal(t, tt.expect, releaseInfo)
		})
	}
}
