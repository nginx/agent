package metric_gen

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator(t *testing.T) {
	g, err := NewGenerator(Config{
		UniquePercent:       50,
		DimensionSize:       3,
		MetricSetsPerMinute: 6000,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	output := make(chan *Message)
	go func() {
		err := g.Generate(ctx, output)
		require.NoError(t, err)
	}()

	for i := 0; i < 10; i++ {
		result := <-output
		splitResult := strings.Split(result.Message, ";")
		splitResult = splitResult[:len(splitResult)-1]

		for _, metricSet := range splitResult {
			msgSplit := strings.Split(metricSet, " ")

			var placement int
			if msgSplit[0] != "" {
				placement = HTTPPlacement
			} else {
				placement = TCPPlacement
			}

			for i, msgField := range msgSplit {
				if fieldOrder[i].Placement == placement || fieldOrder[i].Placement == allPlacement {
					assert.NotEqual(t, msgField, "")

					if fieldOrder[i].Type == stringType || fieldOrder[i].Type == setStringType {
						assert.Len(t, strings.Split(msgField, "\""), 3)
					}
					if fieldOrder[i].Type == intDimensionType {
						assert.Len(t, strings.Split(msgField, "\""), 1)
					}
				}
			}

		}
	}

	cancel()
}

func TestGenerator_Simple(t *testing.T) {
	g, err := NewGenerator(Config{
		UniquePercent:       50,
		DimensionSize:       3,
		MetricSetsPerMinute: 6000,
	}, WithSimpleSet(true))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	output := make(chan *Message)
	go func() {
		err := g.Generate(ctx, output)
		require.NoError(t, err)
	}()

	for i := 0; i < 10; i++ {
		result := <-output
		splitResult := strings.Split(result.Message, ";")
		splitResult = splitResult[:len(splitResult)-1]

		for _, metricSet := range splitResult {
			msgSplit := strings.Split(metricSet, " ")

			var placement int
			if msgSplit[0] != "" {
				placement = HTTPPlacement
			} else {
				placement = TCPPlacement
			}

			for i, msgField := range msgSplit {
				if fieldOrder[i].Placement == placement || fieldOrder[i].Placement == allPlacement {
					assert.NotEqual(t, msgField, "")

					if fieldOrder[i].Type == stringType || fieldOrder[i].Type == setStringType {
						assert.Len(t, strings.Split(msgField, "\""), 3, fmt.Sprintf("`%v` - %v", msgField, fieldOrder[i].Name))
					}
					if fieldOrder[i].Type == intDimensionType {
						assert.Len(t, strings.Split(msgField, "\""), 1, fieldOrder[i].Name)
					}
				}
			}

		}
	}

	cancel()
}

func TestDifferentFields(t *testing.T) {
	HTTPMessage := "\"/\" c8 \"GET\" 1 4e 173 \"DEV_top_http\" \"app_www_variable\" \"localFS_app_www_variable\" 0100007fffff00000000000000000000 4   \"localhost\" 0 0 0 0     \"PASSED\"  \"localFS_gw_app_www_variable\"      0   \"web\" \"http\" 4e 173"
	TCPMessage := "   1   \"env1\" \"app1\" \"tcpcomp1\" 0100007fffff00000000000000000000               \"tcpgw\"        0 \"tcp-udp\" \"tcp\" 0 0"

	splitHTTP := strings.Split(HTTPMessage, " ")
	splitTCP := strings.Split(TCPMessage, " ")

	var ALLFields []string
	var TCPFields []string
	var HTTPFields []string
	var NONEFields []string

	for i, f := range fieldOrder {
		if splitTCP[i] != "" && splitHTTP[i] != "" {
			ALLFields = append(ALLFields, f.Name)
			continue
		}
		if splitTCP[i] != "" {
			TCPFields = append(TCPFields, f.Name)
		}
		if splitHTTP[i] != "" {
			HTTPFields = append(HTTPFields, f.Name)
		}
		if splitHTTP[i] == "" && splitTCP[i] == "" {
			NONEFields = append(NONEFields, f.Name)
		}
	}
	fmt.Printf("ALL: %v\nHTTP: %v\nTCP: %v\nNONE: %v\n", ALLFields, HTTPFields, TCPFields, NONEFields)
}
