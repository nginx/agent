package nap

func ReleaseUnmappedBuild(buildVersion string) NAPRelease {
	return NAPRelease{
		NAPPackages:           NAPReleasePackages{},
		NAPCompilerPackages:   NAPReleasePackages{},
		NAPEnginePackages:     NAPReleasePackages{},
		NAPPluginPackages:     NAPReleasePackages{},
		NAPPlusModulePackages: NAPReleasePackages{},
		VersioningDetails: NAPVersioningDetails{
			NAPBuild:   buildVersion,
			NAPRelease: buildVersion,
		},
	}
}
