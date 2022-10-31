package nap

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nginx/agent/v2/src/core"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultOptNAPDir      = "/opt/app_protect"
	DefaultNMSCompilerDir = "/opt/nms-nap-compiler"
	compilerDirPrefix     = "app_protect-"

	dirPerm = 0755
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
func NewNginxAppProtect(napDir, napSymLinkDir string) (*NginxAppProtect, error) {
	nap := &NginxAppProtect{
		Status:                  "",
		Release:                 NAPRelease{},
		AttackSignaturesVersion: "",
		ThreatCampaignsVersion:  "",
		napDir:                  napDir,
		napSymLinkDir:           napSymLinkDir,
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
	// Initial symlink sync
	if nap.Release.VersioningDetails.NAPRelease != "" {
		err := nap.removeExistingNAPSymlinks()
		if err != nil {
			log.Errorf("Got the following error clearing directory (%s) of existing NAP symlinks - %v", nap.napSymLinkDir, err)
		}

		err = nap.syncSymLink("", nap.Release.VersioningDetails.NAPRelease)
		if err != nil {
			log.Errorf("Error occurred while performing initial sync for NAP symlink  - %v", err)
		}
	}

	for {
		newNap, err := NewNginxAppProtect(nap.napDir, nap.napSymLinkDir)
		if err != nil {
			log.Errorf("The following error occurred while monitoring NAP - %v", err)
			time.Sleep(pollInterval)
			continue
		}

		newNAPReport := newNap.GenerateNAPReport()

		// Check if there has been any change in the NAP report
		if nap.napReportIsEqual(newNAPReport) {
			log.Infof("No change in NAP detected... Checking NAP again in %v seconds", pollInterval.Seconds())
			time.Sleep(pollInterval)
			continue
		}

		// Get NAP report before values are updated to allow sending previous NAP report
		// values via the channel
		previousReport := nap.GenerateNAPReport()
		log.Infof("Change in NAP detected... \nPrevious: %+v\nUpdated: %+v\n", previousReport, newNAPReport)

		err = nap.syncSymLink(nap.Release.VersioningDetails.NAPRelease, newNAPReport.NAPVersion)
		if err != nil {
			log.Errorf("Got the following error syncing NAP symlink - %v", err)
			time.Sleep(pollInterval)
			continue
		}

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

// syncSymLink determines if the symlink for the NAP installation needs to be updated
// or not and performs the necessary actions to do so.
func (nap *NginxAppProtect) syncSymLink(previousVersion, newVersion string) error {
	oldSymLink := filepath.Join(nap.napSymLinkDir, compilerDirPrefix+previousVersion)
	nmsCompilerSymLinkDir := filepath.Join(nap.napSymLinkDir, compilerDirPrefix+newVersion)

	switch {
	// Same version no need for updating symlink
	case previousVersion == newVersion:
		return nil

	// NAP was removed
	case newVersion == "":
		return nap.removeSymlink(oldSymLink)
	}

	// Check if the necessary directory exists
	_, err := os.Stat(nap.napSymLinkDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(nap.napSymLinkDir, dirPerm)
		if err != nil {
			return err
		}
		log.Debugf("Successfully create the directory %s for creating NAP symlink", nap.napSymLinkDir)
	} else if err != nil {
		return err
	}

	// Check if the symlink exists b/c it needs to be removed in order to update it if
	// that's the case
	log.Debugf("Attempting to create symlink %s -> %s", nmsCompilerSymLinkDir, nap.napDir)
	err = nap.removeSymlink(nmsCompilerSymLinkDir)
	if err != nil {
		return err
	}
	err = os.Symlink(nap.napDir, nmsCompilerSymLinkDir)
	if err != nil {
		return err
	}

	// Once new symlink is created remove old one if it exists
	log.Debugf("Deleting previous NAP symlink %s -> %s", oldSymLink, nap.napDir)
	return nap.removeSymlink(oldSymLink)
}

// removeSymlink removes the specified symlink if it exists. If it doesn't exist
// no error is returned.
func (nap *NginxAppProtect) removeSymlink(symLinkPath string) error {
	_, err := os.Lstat(symLinkPath)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	default:
		return os.Remove(symLinkPath)
	}
}

// removeExistingNAPSymlinks walks the NAP symlink directory and removes any existing
// NAP symlinks found in the directory.
func (nap *NginxAppProtect) removeExistingNAPSymlinks() error {
	// Check if the necessary directory exists
	_, err := os.Stat(nap.napSymLinkDir)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err = filepath.WalkDir(nap.napSymLinkDir, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}

		// If it doesn't contain the compiler symlink dir prefix skip the file
		if !strings.Contains(d.Name(), compilerDirPrefix) {
			return nil
		}

		return os.Remove(filepath.Join(nap.napSymLinkDir, d.Name()))
	})

	return err
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
	log.Debugf("Checking for the required NAP files - %v\n", requiredFiles)
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
		log.Debugf("The following required NAP process(es) couldn't be found: %v", missingProcesses)
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
