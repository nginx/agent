/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"

	log "github.com/sirupsen/logrus"
)

// UpdateMetadata retrieves the NAP content from the config and writes
// the metadata
func UpdateMetadata(
	cfg *proto.NginxConfig,
	appProtectWAFDetails *proto.AppProtectWAFDetails,
) error {
	previousPrecompiledPublication := false
	previousMeta := Metadata{}

	// Read NAP metadata
	data, err := os.ReadFile(appProtectWAFDetails.GetWafLocation())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &previousMeta); err != nil {
			return err
		}
		previousPrecompiledPublication = previousMeta.PrecompiledPublication
	}

	// Write the metadata if precomp publication is false, or
	// when precomp publication toggles to true.
	// If toggled, write metadata once more then the publisher
	// will send metadata thereafter.
	if previousPrecompiledPublication && appProtectWAFDetails.GetPrecompiledPublication() {
		return nil
	}

	policies, profiles := sdk.GetAppProtectPolicyAndSecurityLogFiles(cfg)

	policyBundles := []*BundleMetadata{}
	profileBundles := []*BundleMetadata{}

	for _, policy := range policies {
		bundle := &BundleMetadata{
			Name: policy,
		}
		policyBundles = append(policyBundles, bundle)
	}
	for _, profile := range profiles {
		bundle := &BundleMetadata{
			Name: profile,
		}
		profileBundles = append(profileBundles, bundle)
	}

	metadata := &Metadata{
		NapVersion:                       appProtectWAFDetails.GetWafVersion(),
		PrecompiledPublication:           appProtectWAFDetails.GetPrecompiledPublication(),
		AttackSignatureRevisionTimestamp: appProtectWAFDetails.GetAttackSignaturesVersion(),
		ThreatCampaignRevisionTimestamp:  appProtectWAFDetails.GetThreatCampaignsVersion(),
		Policies:                         policyBundles,
		Profiles:                         profileBundles,
	}

	// Check if metadata changed, don't need to write if unchanged
	if metadataAreEqual(&previousMeta, metadata) {
		return nil
	}

	m, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// Make dir if not exists
	directory := filepath.Dir(appProtectWAFDetails.GetWafLocation())
	_, err = os.Stat(directory)
	if os.IsNotExist(err) {
		err = os.MkdirAll(directory, 0755)
		if err != nil {
			return err
		}
	}

	log.Debugf("Writing NAP Metadata %s", m)
	return os.WriteFile(appProtectWAFDetails.GetWafLocation(), m, 0644)
}

// metadataAreEqual compares the metadata for equality
func metadataAreEqual(oldMeta, newMeta *Metadata) bool {
	if oldMeta.NapVersion != newMeta.NapVersion {
		return false
	}
	if oldMeta.PrecompiledPublication != newMeta.PrecompiledPublication {
		return false
	}
	if oldMeta.AttackSignatureRevisionTimestamp != newMeta.AttackSignatureRevisionTimestamp {
		return false
	}
	if oldMeta.ThreatCampaignRevisionTimestamp != newMeta.ThreatCampaignRevisionTimestamp {
		return false
	}
	if len(oldMeta.Policies) != len(newMeta.Policies) {
		return false
	}
	if len(oldMeta.Profiles) != len(newMeta.Profiles) {
		return false
	}
	for _, oldPolicy := range oldMeta.Policies {
		found := false
		for _, newPolicy := range newMeta.Policies {
			if newPolicy.Name == oldPolicy.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, oldProfile := range oldMeta.Profiles {
		found := false
		for _, newProfile := range newMeta.Profiles {
			if newProfile.Name == oldProfile.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
