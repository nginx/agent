package nap

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/v2/src/core"
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
func NewNginxAppProtect(optDirPath, symLinkDir string) (*NginxAppProtect, error) {
	nap := &NginxAppProtect{
		Status:                  "",
		Release:                 NAPRelease{},
		AttackSignaturesVersion: "",
		ThreatCampaignsVersion:  "",
		optDirPath:              optDirPath,
		symLinkDir:              symLinkDir,
	}

	// Get status of NAP on the system
	status, err := napStatus(requiredNAPFiles)
	if err != nil {
		return nil, err
	}

	// Get the release version of NAP on the system if NAP is installed
	var napRelease *NAPRelease
	if status != MISSING {
		napRelease, err = installedNAPRelease(NAP_VERSION_FILE)
		if err != nil {
			return nil, err
		}
	}

	// Update the NAP object with the values from NAP on the system
	nap.Status = status.String()
	if napRelease != nil {
		nap.Release = *napRelease
	}

	return nap, nil
}

// Monitor starts a goroutine responsible for monitoring the system for any NAP related
// changes and communicates those changes with a report message sent via the channel this
// function returns. Additionally, if any changes are detected the NAP object that called
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
		err := nap.syncSymLink("", nap.Release.VersioningDetails.NAPRelease)
		if err != nil {
			log.Errorf("Error occurred while performing initial sync for NAP symlink  - %v", err)
		}
	}

	ticker := time.NewTicker(pollInterval)

	for {
		select {
		case <-ticker.C:
			newNap, err := NewNginxAppProtect(nap.optDirPath, nap.symLinkDir)
			if err != nil {
				log.Errorf("The following error occurred while monitoring NAP - %v", err)
				break
			}

			newNAPReport := newNap.GenerateNAPReport()

			// Check if there has been any change in the NAP report
			if nap.napReportIsEqual(newNAPReport) {
				log.Debugf("No change in NAP detected... Checking NAP again in %v seconds", pollInterval.Seconds())
				break
			}

			// Get NAP report before values are updated to allow sending previous NAP report
			// values via the channel
			previousReport := nap.GenerateNAPReport()
			log.Infof("Change in NAP detected... \nPrevious: %+v\nUpdated: %+v\n", previousReport, newNAPReport)

			err = nap.syncSymLink(nap.Release.VersioningDetails.NAPRelease, newNAPReport.NAPVersion)
			if err != nil {
				log.Errorf("Got the following error syncing NAP symlink - %v", err)
				break
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
		}

	}
}

// syncSymLink determines if the symlink for the NAP installation needs to be updated
// or not and performs the necessary actions to do so.
func (nap *NginxAppProtect) syncSymLink(previousVersion, newVersion string) error {
	oldSymLink := filepath.Join(nap.symLinkDir, compilerDirPrefix+previousVersion)
	nmsCompilerSymLinkDir := filepath.Join(nap.symLinkDir, compilerDirPrefix+newVersion)

	if previousVersion == newVersion {
		// Same version no need for updating symlink
		return nil
	} else if newVersion == "" {
		// NAP was removed so remove all NAP symlinks
		return nap.removeNAPSymlinks("")
	}

	// Check if the necessary directory exists
	_, err := os.Stat(nap.symLinkDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(nap.symLinkDir, dirPerm)
		if err != nil {
			return err
		}
		log.Debugf("Successfully create the directory %s for creating NAP symlink", nap.symLinkDir)
	} else if err != nil {
		return err
	}

	// Remove existing NAP symlinks except for currently used one, b/c if we're updating a
	// symlink that already exists then we need to remove then create the updated one.
	err = nap.removeNAPSymlinks(previousVersion)
	if err != nil {
		return err
	}

	// Create new symlink
	log.Debugf("Creating symlink %s -> %s", nmsCompilerSymLinkDir, nap.optDirPath)
	err = os.Symlink(nap.optDirPath, nmsCompilerSymLinkDir)
	if err != nil {
		return err
	}

	// Once new symlink is created remove old one if it exists
	log.Debugf("Deleting previous NAP symlink %s -> %s", oldSymLink, nap.optDirPath)
	return nap.removeNAPSymlinks(newVersion)
}

// removeNAPSymlinks walks the NAP symlink directory and removes any existing NAP
// symlinks found in the directory except for ones that match to ignore pattern.
func (nap *NginxAppProtect) removeNAPSymlinks(symlinkPatternToIgnore string) error {
	// Check if the necessary directory exists
	_, err := os.Stat(nap.symLinkDir)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err = filepath.WalkDir(nap.symLinkDir, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}

		// If it doesn't contain the compiler symlink dir prefix skip the file
		if !strings.Contains(d.Name(), compilerDirPrefix) || strings.Contains(d.Name(), symlinkPatternToIgnore) {
			return nil
		}

		return os.Remove(filepath.Join(nap.symLinkDir, d.Name()))
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
// it's possible NAP might be installed but an error verifying its installation has occurred.
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
	installed, err := napInstalled(requiredFiles)
	if !installed && err == nil {
		return MISSING, nil
	} else if err != nil {
		return UNDEFINED, err
	}

	// It's installed, but is running?
	running, err := napRunning()
	if !running && err == nil {
		return INSTALLED, nil
	} else if err != nil {
		return UNDEFINED, err
	}

	return RUNNING, nil
}
