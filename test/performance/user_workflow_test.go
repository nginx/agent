package performance

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/sdk/v2/zip"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/stretchr/testify/require"
)

var (
	largeConfigFiles = []string{
		"../testdata/configs/bigger/1k.conf",
		"../testdata/configs/bigger/2k.conf",
		"../testdata/configs/bigger/3k.conf",
		"../testdata/configs/bigger/10k.conf",
	}

	deleteDirectories = []string{
		"../testdata/configs/bigger/ssl",
		"../testdata/configs/bigger/conf",
	}
)

// BenchmarkNginxConfig benchmarks the generation of the NginxConfig struct
func BenchmarkNginxConfig(b *testing.B) {
	err := generateCertificate()
	if err != nil {
		b.Error(err)
	}
	for _, v := range largeConfigFiles {
		func(confFile string) {
			b.Run(confFile, func(bb *testing.B) {
				bb.ReportAllocs()

				allowedDirs := map[string]struct{}{}
				allowedDirs["../testdata/configs/bigger/ssl/"] = struct{}{}

				var nginxConfig *proto.NginxConfig
				var err error
				for n := 0; n < b.N; n++ {
					nginxConfig, err = sdk.GetNginxConfig(confFile, "", "", allowedDirs)
				}
				require.NoError(bb, err)
				require.NotNil(bb, nginxConfig, "Generated nginxConfig struct should not be nil")
			})
		}(v)
	}

	err = cleanUpCertificates()
	if err != nil {
		b.Error(err)
	}
}

// BenchmarkGetConfigFiles benchmarks unpacking the nginxConfig struct and getting the
// config files and aux files ** note ssl files not added to auxFiles so it's always nil
func BenchmarkGetConfigFiles(b *testing.B) {
	err := generateCertificate()
	if err != nil {
		b.Error(err)
	}
	configs, err := genConfig()
	if err != nil {
		b.Error(err)
	}

	for _, v := range configs {
		func(config *proto.NginxConfig) {
			b.Run("GetConfigFiles", func(bb *testing.B) {
				bb.ReportAllocs()
				var confFiles []*proto.File
				var auxFiles []*proto.File
				var err error
				for n := 0; n < b.N; n++ {
					confFiles, auxFiles, err = sdk.GetNginxConfigFiles(config)
				}
				require.NoError(bb, err)
				conf := fmt.Sprintf("Generated config Files for %v should not be nil", config.GetDirectoryMap().GetDirectories()[0].GetFiles()[0].GetName())
				confAux := fmt.Sprintf("Generated auxillary files for %v should not be nil", config.GetDirectoryMap().GetDirectories()[0].GetFiles()[0].GetName())

				require.NotNil(bb, confFiles, conf)
				require.NotNil(bb, auxFiles, confAux)
			})
		}(v)
	}
	err = cleanUpCertificates()
	if err != nil {
		b.Error(err)
	}
}

// BenchmarkReadConfig benchmarks reading nginx config from disk and generating the nginxConfig struct
func BenchmarkReadConfig(b *testing.B) {
	err := generateCertificate()
	if err != nil {
		b.Error(err)
	}
	for _, v := range largeConfigFiles {
		allowedDirs := map[string]struct{}{}
		allowedDirs["../testdata/configs/bigger/ssl/"] = struct{}{}
		binary := core.NewNginxBinary(nil, &config.Config{AllowedDirectoriesMap: allowedDirs})

		func(config string) {
			testName := strings.Join([]string{"Read Config ", config}, "")
			b.Run(testName, func(bb *testing.B) {
				bb.ReportAllocs()
				var err error
				var nginxConfig *proto.NginxConfig
				for n := 0; n < b.N; n++ {
					nginxConfig, err = binary.ReadConfig(config, "", "")
				}
				require.NoError(bb, err)
				require.NotNil(bb, nginxConfig, "NginxConfig read in should not be nil")
			})
		}(v)
	}
	err = cleanUpCertificates()
	if err != nil {
		b.Error(err)
	}
}

// BenchmarkZipConfig benchmarks the zipping of the contents of nginx config into ZippedFile struct
func BenchmarkZipConfig(b *testing.B) {
	err := generateCertificate()
	if err != nil {
		b.Error(err)
	}
	for _, v := range largeConfigFiles {
		func(config string) {
			testName := strings.Join([]string{"Zip config", config}, "")
			b.Run(testName, func(bb *testing.B) {
				bb.ReportAllocs()
				var conf *zip.Writer
				var err1, err2 error
				var zipped *proto.ZippedFile

				for n := 0; n < b.N; n++ {
					conf, err1 = zip.NewWriter(config)
					zipped, err2 = conf.Proto()
				}
				require.NoError(bb, err1)
				require.NoError(bb, err2)
				require.NotNil(bb, zipped, "Zipped Config should not be nil")
			})
		}(v)
	}
	err = cleanUpCertificates()
	if err != nil {
		b.Error(err)
	}
}

// BenchmarkUnZipConfig benchmarks the unzipping of zipped content
func BenchmarkUnZipConfig(b *testing.B) {
	err := generateCertificate()
	if err != nil {
		b.Error(err)
	}
	zippedConfigs := []*proto.ZippedFile{}
	for _, v := range largeConfigFiles {
		conf, err := zip.NewWriter(v)
		if err != nil {
			b.Error(err)
		}
		zipped, err := conf.Proto()
		if err != nil {
			b.Error(err)
		}

		zippedConfigs = append(zippedConfigs, zipped)
	}

	for _, v := range zippedConfigs {
		func(config *proto.ZippedFile) {
			b.Run("", func(bb *testing.B) {
				bb.ReportAllocs()
				var files []*proto.File
				var err error
				for n := 0; n < b.N; n++ {
					files, err = zip.UnPack(config)
				}
				require.NoError(bb, err)
				require.NotNil(bb, files, "Unzipped Files should not be nil")
			})
		}(v)
	}
	err = cleanUpCertificates()
	if err != nil {
		b.Error(err)
	}
}

func genConfig() ([]*proto.NginxConfig, error) {
	configs := []*proto.NginxConfig{}
	for _, confFile := range largeConfigFiles {
		allowedDirs := map[string]struct{}{}
		allowedDirs["../testdata/configs/bigger/ssl/"] = struct{}{}
		nginxConfig, err := sdk.GetNginxConfig(confFile, "", "", allowedDirs)
		if err != nil {
			return nil, err
		}
		configs = append(configs, nginxConfig)
	}
	return configs, nil
}

// generateCertificate generates the certificates referenced in the nginx config
func generateCertificate() error {
	for i := 1; i <= 3; i++ {
		agentVersion := fmt.Sprintf("agent%v", i)
		filename := fmt.Sprintf("%v.local", agentVersion)
		cmd := exec.Command("../../scripts/tls/gen_cnf.sh", "ca", "--cn", filename, "--state", "Cork", "--locality", "Cork", "--org", "NGINX", "--country", "IE", "--out", "../testdata/configs/bigger/conf")

		err := cmd.Run()
		if err != nil {
			return err
		}

		cmd1 := exec.Command("../../scripts/tls/gen_cert.sh", "ca", "--config", "../testdata/configs/bigger/conf/ca.cnf", "--out", "../testdata/configs/bigger/ssl")
		err = cmd1.Run()
		if err != nil {
			return err
		}

		newCrtFile := fmt.Sprintf("../testdata/configs/bigger/ssl/%v.crt", agentVersion)
		newkeyFile := fmt.Sprintf("../testdata/configs/bigger/ssl/%v.key", agentVersion)

		err = os.Rename("../testdata/configs/bigger/ssl/ca.crt", newCrtFile)
		if err != nil {
			return err
		}

		err = os.Rename("../testdata/configs/bigger/ssl/ca.key", newkeyFile)
		if err != nil {
			return err
		}
	}

	filename := "test.local"
	cmd := exec.Command("../../scripts/tls/gen_cnf.sh", "ca", "--cn", filename, "--state", "Cork", "--locality", "Cork", "--org", "NGINX", "--country", "IE", "--out", "../testdata/configs/bigger/conf")

	err := cmd.Run()
	if err != nil {
		return err
	}

	cmd1 := exec.Command("../../scripts/tls/gen_cert.sh", "ca", "--config", "../testdata/configs/bigger/conf/ca.cnf", "--out", "../testdata/configs/bigger/ssl")
	err = cmd1.Run()
	if err != nil {
		return err
	}

	return nil
}

func cleanUpCertificates() error {
	for _, dir := range deleteDirectories {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	return nil
}
