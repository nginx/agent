/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

const (
	NAP_VERSION_FILE         = "/opt/app_protect/VERSION"
	BD_SOCKET_PLUGIN_PATH    = "/usr/share/ts/bin/bd-socket-plugin"
	BD_SOCKET_PLUGIN_PROCESS = "bd-socket-plugin"

	// TODO: Rather than using the update yaml files for attack signatures and threat
	// campaigns we should use the version files. We're currently using the update files
	// to determine the versions because the version files for attack signatures and threat
	// campaigns are write protected files this means when we do a "purge" on the packages
	// the packages seem to be removed but their version files are not. This causes an issue
	// because we rely on the version files existing or not existing to determine if those
	// packages are installed and get the versions. This means when the packages are removed
	// but version files aren't then we're reporting back that these packages are installed and
	// report their versions.
	// ATTACK_SIGNATURE_VERSION_FILE = "/opt/app_protect/var/update_files/signatures/version"
	// THREAT_CAMPAIGN_VERSION_FILE  = "/opt/app_protect/var/update_files/threat_campaigns/version"
	ATTACK_SIGNATURES_UPDATE_FILE = "/opt/app_protect/var/update_files/signatures/signature_update.yaml"
	THREAT_CAMPAIGNS_UPDATE_FILE  = "/opt/app_protect/var/update_files/threat_campaigns/threat_campaign_update.yaml"

	APP_PROTECT_METADATA_FILE_PATH = "/etc/nms/app_protect_metadata.json"
)
