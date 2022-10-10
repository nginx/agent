package nap

import (
	"fmt"
	"time"

	"github.com/nginx/agent/v2/src/core"
)

var (
	requiredNAPFiles    = []string{BD_SOCKET_PLUGIN_PATH, NAP_VERSION_FILE}
	requireNAPProcesses = []string{BD_SOCKET_PLUGIN_PROCESS}
	processCheckFunc    = core.CheckForProcesses
)

// NewNginxAppProtect returns the object NginxAppProtect, which contains information related
// to the Nginx App Protect installed on the system. If Nginx App Protect is NOT installed on
// the system then a NginxAppProtect object is still returned, the status field will be set
// as MISSING and all other fields will be blank.
func NewNginxAppProtect() (*NginxAppProtect, error) {

	nap := &NginxAppProtect{
		Status:                  "",
		Release:                 NAPRelease{},
		AttackSignaturesVersion: "",
		ThreatCampaignsVersion:  "",
	}

	// Get status of NAP on the system
	napStatus, err := napStatus(requiredNAPFiles)
	if err != nil {
		return nil, err
	}

	// Get the release version of NAP on the system if NAP is installed
	var napRelease *NAPRelease
	if napStatus != MISSING {
		napRelease, err = installedNAPRelease(NAP_VERSION_FILE)
		if err != nil {
			return nil, err
		}
	}

	// Get attack signatures version
	attackSigsVersion, err := getAttackSignaturesVersion(ATTACK_SIGNATURES_UPDATE_FILE)
	if err != nil && err.Error() != fmt.Sprintf(FILE_NOT_FOUND, ATTACK_SIGNATURES_UPDATE_FILE) {
		return nil, err
	}

	// Get threat campaigns version
	threatCampaignsVersion, err := getThreatCampaignsVersion(THREAT_CAMPAIGNS_UPDATE_FILE)
	if err != nil && err.Error() != fmt.Sprintf(FILE_NOT_FOUND, THREAT_CAMPAIGNS_UPDATE_FILE) {
		return nil, err
	}

	// Update the NAP object with the values from NAP on the system
	nap.Status = napStatus.String()
	nap.AttackSignaturesVersion = attackSigsVersion
	nap.ThreatCampaignsVersion = threatCampaignsVersion
	if napRelease != nil {
		nap.Release = *napRelease
	}

	return nap, nil
}

// Monitor starts a goroutine responsible for monitoring the system for any NAP related
// changes and communicates those changes with a report message sent via the channel this
// function returns. Additionally if any changes are detected the NAP object that called
// this monitoring function will have its attributes updated to the new changes. Here are
// examples of NAP changes that would be detected and communicated:
//   - NAP installed/version changed
//   - NAP started running
//   - NAP stopped running
//   - NAP version changed
//   - Attack signature installed/version changed
//   - Threat campaign installed/version changed
func (nap *NginxAppProtect) Monitor(pollInterval time.Duration) chan NAPReportBundle {
	msgChannel := make(chan NAPReportBundle)
	go nap.monitor(msgChannel, pollInterval)
	return msgChannel
}

// monitor checks the system for any NAP related changes and communicates those changes with
// a report message sent via the channel provided to it.
func (nap *NginxAppProtect) monitor(msgChannel chan NAPReportBundle, pollInterval time.Duration) {
	for {
		newNap, err := NewNginxAppProtect()
		if err != nil {
			logger.Errorf("The following error occurred while monitoring NAP - %v", err)
			time.Sleep(pollInterval)
			continue
		}

		newNAPReport := newNap.GenerateNAPReport()

		// Check if there has been any change in the NAP report
		if nap.napReportIsEqual(newNAPReport) {
			logger.Debugf("No change in NAP detected... Checking NAP again in %v seconds", pollInterval.Seconds())
			time.Sleep(pollInterval)
			continue
		}

		// Get NAP report before values are updated to allow sending previous NAP report
		// values via the channel
		previousReport := nap.GenerateNAPReport()
		logger.Debugf("Change in NAP detected... \nPrevious: %+v\nUpdated: %+v\n", previousReport, newNAPReport)

		// Update the current NAP values since there was a change
		nap.Status = newNap.Status
		nap.Release = newNap.Release
		nap.AttackSignaturesVersion = newNap.AttackSignaturesVersion
		nap.ThreatCampaignsVersion = newNap.ThreatCampaignsVersion

		// Send the update message through the channel
		msgChannel <- NAPReportBundle{
			PreviousReport: previousReport,
			UpdatedReport:  newNAPReport,
		}

		time.Sleep(pollInterval)
	}
}

// GenerateNAPReport generates a NAPReport based off the NAP object calling
// this function. This means the report contains the values from the NAP object which
// COULD be different from the current system NAP values if the NAP object that called this
// function has NOT called the Monitor function that is responsible for updating its values
// to be in sync with the current system NAP values.
func (nap *NginxAppProtect) GenerateNAPReport() NAPReport {
	return NAPReport{
		NAPVersion:              nap.Release.VersioningDetails.NAPRelease,
		Status:                  nap.Status,
		AttackSignaturesVersion: nap.AttackSignaturesVersion,
		ThreatCampaignsVersion:  nap.ThreatCampaignsVersion,
	}
}

// napReportIsEqual determines if the nap report being passed into this function is equal
// to the napReport that the NAP object calling this function produces.
func (nap *NginxAppProtect) napReportIsEqual(incomingNAPReport NAPReport) bool {
	currentNAPReport := nap.GenerateNAPReport()
	return (currentNAPReport.NAPVersion == incomingNAPReport.NAPVersion) &&
		(currentNAPReport.Status == incomingNAPReport.Status) &&
		(currentNAPReport.AttackSignaturesVersion == incomingNAPReport.AttackSignaturesVersion) &&
		(currentNAPReport.ThreatCampaignsVersion == incomingNAPReport.ThreatCampaignsVersion)
}

// napInstalled determines if NAP is installed on the system. If NAP is NOT installed on the
// system then the bool will be false and the error will be nil, if the error is not nil then
// it's possible NAP might be installed but an error verifying it's installation has occurred.
func napInstalled(requiredFiles []string) (bool, error) {
	logger.Debugf("Checking for the required NAP files - %v\n", requiredFiles)
	return core.FilesExists(requiredFiles)
}

// napRunning determines if Nginx App Protect is running on the system or not. If NAP is
// NOT running then the bool will be false and the error will be nil, if an error occurred
// the bool will be false and the error will not be nil.
func napRunning() (bool, error) {
	// Check if NAP is running
	missingProcesses, err := processCheckFunc(requireNAPProcesses)
	if err != nil {
		return false, err
	}

	if len(missingProcesses) != 0 {
		logger.Debugf("The following required NAP process(es) couldn't be found: %v", missingProcesses)
		return false, nil
	}

	return true, nil
}

// napStatus gets the current status of NAP on the system. The status will be one of the
// following:
// - MISSING
// - INSTALLED
// - RUNNING
func napStatus(requiredFiles []string) (Status, error) {

	// Check if NAP is installed
	napInstalled, err := napInstalled(requiredFiles)
	if !napInstalled && err == nil {
		return MISSING, nil
	} else if err != nil {
		return UNDEFINED, err
	}

	// It's installed, but is running?
	napRunning, err := napRunning()
	if !napRunning && err == nil {
		return INSTALLED, nil
	} else if err != nil {
		return UNDEFINED, err
	}

	return RUNNING, nil
}
