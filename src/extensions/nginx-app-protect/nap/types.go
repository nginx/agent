/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

// Status is an Enum that represents the status of NAP.
type Status int

// NginxAppProtect is the object representation of Nginx App Protect, it contains information
// related to the Nginx App Protect on the system.
type NginxAppProtect struct {
	Status                  string
	Release                 NAPRelease
	AttackSignaturesVersion string
	ThreatCampaignsVersion  string
	optDirPath              string
	symLinkDir              string
}

// NAPReport is a collection of information on the current systems NAP details.
type NAPReport struct {
	Status                  string
	NAPVersion              string
	AttackSignaturesVersion string
	ThreatCampaignsVersion  string
}

// NAPReportBundle is meant to capture the NAPReport before an update
// has occurred on NAP as well as capture the NAPReport after an update has
// occurred on NAP.
type NAPReportBundle struct {
	PreviousReport NAPReport
	UpdatedReport  NAPReport
}

// NAPRelease captures information like specific packages and versions for a specific
// NAP release.
type NAPRelease struct {
	NAPPackages           NAPReleasePackages   `json:"nap-packages,omitempty"`
	NAPCompilerPackages   NAPReleasePackages   `json:"nap-compiler-packages,omitempty"`
	NAPEnginePackages     NAPReleasePackages   `json:"nap-engine-packages,omitempty"`
	NAPPluginPackages     NAPReleasePackages   `json:"nap-plugin-packages,omitempty"`
	NAPPlusModulePackages NAPReleasePackages   `json:"nap-plus-module-packages,omitempty"`
	VersioningDetails     NAPVersioningDetails `json:"versioning-details,omitempty"`
}

// NAPReleasePackages represents the package needed on a specific OS from the supported
// OSs for a specific release package.
type NAPReleasePackages struct {
	Alpine310    string `json:"alpine-3.10,omitempty"`
	AmazonLinux2 string `json:"amazon-linux-2,omitempty"`
	Centos7      string `json:"centos-7,omitempty"`
	Debian9      string `json:"debian-9,omitempty"`
	Debian10     string `json:"debian-10,omitempty"`
	Redhat7      string `json:"redhat-7,omitempty"`
	Redhat8      string `json:"redhat-8,omitempty"`
	Ubuntu1804   string `json:"ubuntu-18.04,omitempty"`
	Ubuntu2004   string `json:"ubuntu-20.04,omitempty"`
}

// NAPVersioningDetails provides the version information for packages related to NAP.
type NAPVersioningDetails struct {
	NAPBuild      string `json:"nap-build,omitempty"`
	NAPCompiler   string `json:"nap-compiler,omitempty"`
	NAPEngine     string `json:"nap-engine,omitempty"`
	NAPPlugin     string `json:"nap-plugin,omitempty"`
	NAPPlusModule string `json:"nap-plus-module,omitempty"`
	NAPRelease    string `json:"nap-release"`
	NginxPlus     string `json:"nginx-plus,omitempty"`
}

// NAPReleaseMap is a mapping object meant to capture a specific NAP Release version as
// the key and NAP Release information as the value.
type NAPReleaseMap struct {
	ReleaseMap map[string]NAPRelease `json:"releases"`
}

type Metadata struct {
	NapVersion                       string            `json:"napVersion"`
	GlobalStateFileName              string            `json:"globalStateFileName"`
	GlobalStateFileUID               string            `json:"globalStateFileUID"`
	AttackSignatureRevisionTimestamp string            `json:"attackSignatureRevisionTimestamp,omitempty"`
	AttackSignatureUID               string            `json:"attackSignatureUID,omitempty"`
	ThreatCampaignRevisionTimestamp  string            `json:"threatCampaignRevisionTimestamp,omitempty"`
	ThreatCampaignUID                string            `json:"threatCampaignUID,omitempty"`
	Policies                         []*BundleMetadata `json:"policyMetadata,omitempty"`
	Profiles                         []*BundleMetadata `json:"logProfileMetadata,omitempty"`
}

type BundleMetadata struct {
	Name              string `json:"name"`
	UID               string `json:"uid"`
	RevisionTimestamp int64  `json:"revisionTimestamp"`
}
