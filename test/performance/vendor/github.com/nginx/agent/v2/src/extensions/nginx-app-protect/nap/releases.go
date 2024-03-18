/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

func ReleaseUnmappedBuild(version, release string) NAPRelease {
	return NAPRelease{
		NAPPackages:           NAPReleasePackages{},
		NAPCompilerPackages:   NAPReleasePackages{},
		NAPEnginePackages:     NAPReleasePackages{},
		NAPPluginPackages:     NAPReleasePackages{},
		NAPPlusModulePackages: NAPReleasePackages{},
		VersioningDetails: NAPVersioningDetails{
			NAPBuild:   version,
			NAPRelease: release,
		},
	}
}
