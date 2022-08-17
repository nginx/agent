package core_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/v2/src/core"
	testutils "github.com/nginx/agent/v2/test/utils"
)

const (
	fakeProcOne   = "fake-proc-one"
	fakeProcTwo   = "fake-proc-two"
	fakeProcThree = "fake-proc-three"
)

func TestCheckForProcesses(t *testing.T) {
	testCases := []struct {
		testName        string
		procsToCreate   []string
		procsToCheck    []string
		expMissingProcs []string
	}{
		{
			testName:        "EmptyProcessCheck",
			procsToCreate:   []string{},
			procsToCheck:    []string{},
			expMissingProcs: []string{},
		},
		{
			testName:        "SingleProcessFound",
			procsToCreate:   []string{fakeProcOne},
			procsToCheck:    []string{fakeProcOne},
			expMissingProcs: []string{},
		},
		{
			testName:        "MultipleProcessesFound",
			procsToCreate:   []string{fakeProcOne, fakeProcTwo, fakeProcThree},
			procsToCheck:    []string{fakeProcOne, fakeProcTwo, fakeProcThree},
			expMissingProcs: []string{},
		},
		{
			testName:        "SingleMissingProcess",
			procsToCreate:   []string{fakeProcOne, fakeProcThree},
			procsToCheck:    []string{fakeProcOne, fakeProcTwo, fakeProcThree},
			expMissingProcs: []string{fakeProcTwo},
		},
		{
			testName:        "MultipleMissingProcesses",
			procsToCreate:   []string{fakeProcTwo},
			procsToCheck:    []string{fakeProcOne, fakeProcTwo, fakeProcThree},
			expMissingProcs: []string{fakeProcOne, fakeProcThree},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			killFakeProcesses := testutils.StartFakeProcesses(tc.procsToCreate, "10")
			defer killFakeProcesses()

			missingProcesses, err := core.CheckForProcesses(tc.procsToCheck)

			assert.Equal(t, nil, err, fmt.Sprintf("Expected error to be nil but got %v", err))
			assert.Equal(t, tc.expMissingProcs, missingProcesses, fmt.Sprintf("Expected missing processes to be %v but got %v", tc.expMissingProcs, missingProcesses))
		})
	}
}
