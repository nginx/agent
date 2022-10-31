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
	napDir                  string
	napSymLinkDir           string
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

// napRevisionDateTime is an object used to get the version for attack signatures and
// threat campaigns, as their versions are the same as their revision dates which can be
// captured in their yaml files under the field "revisionDatetime".
type napRevisionDateTime struct {
	RevisionDatetime string `yaml:"revisionDatetime,omitempty"`
	Checksum         string `yaml:"checksum,omitempty"`
	Filename         string `yaml:"filename,omitempty"`
}
