package parser

// import (
// 	"strings"

// 	"github.com/nginx/agent/sdk/v2/proto"
// )

// func parsePHPPoolConf(poolConf string) []*proto.PHPFPMPool {
// 	pools := []*proto.PHPFPMPool{}
// 	var currentPool *proto.PHPFPMPool

// 	poolConfLines := strings.Split(poolConf, "\n")
// 	for _, line := range poolConfLines {
// 		// skip comments, empty lines
// 		if len(line) == 0 || string(line[0]) == ";" {
// 			continue
// 		}

// 		// strip
// 		line = strings.ReplaceAll(line, " ", "")

// 		// first check if this is a new pool declaration
// 		if string(line[0]) == "[" {
// 			// found a new context
// 			if currentPool != nil {
// 				// if this is not the first pool, append
// 				pools = append(pools, currentPool)
// 			}
// 			// and then reset struct
// 			currentPool = &proto.PHPFPMPool{
// 				Includes: []string{},
// 				Name:     getPoolName(line),
// 			}
// 			// and then move to the next line
// 			continue
// 		}

// 		// otherwise, it is a directive within a pool
// 		directive := strings.Split(line, "=")
// 		// this is a malformed directive and should be ignored
// 		if len(directive) != 2 {
// 			continue
// 		}
// 		if directive[0] == "include" {
// 			// found an include
// 			currentPool.Includes = append(currentPool.Includes, directive[1])
// 		} else if directive[0] == "listen" {
// 			// found listen socket
// 			currentPool.Listen = directive[1]
// 			currentPool.Flisten = directive[1]
// 		} else if directive[0] == "pm.status_path" {
// 			// found status page
// 			currentPool.StatusPath = directive[1]
// 		}
// 	}

// 	// append the last active pool we were collecting info on
// 	pools = append(pools, currentPool)

// 	// compute local ids
// 	for _, p := range pools {
// 		p.LocalId = core.GenerateNginxID("%s_%s", p.Name, p.Listen)
// 	}
// 	return pools
// }

// func getPoolName(rawPoolName string) (poolName string) {
// 	poolName = strings.ReplaceAll(rawPoolName, "[", "")
// 	poolName = strings.ReplaceAll(poolName, "]", "")
// 	return poolName
// }
