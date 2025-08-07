// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package files implements utility routines for gathering information about files and their contents.
package files

import (
	"cmp"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/cert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const permissions = 0o600

// FileMeta returns a proto FileMeta struct from a given file path.
func FileMeta(filePath string) (*mpi.FileMeta, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fileHash := GenerateHash(content)
	fileMeta := &mpi.FileMeta{
		Name:         filePath,
		Hash:         fileHash,
		ModifiedTime: timestamppb.New(fileInfo.ModTime()),
		Permissions:  Permissions(fileInfo.Mode()),
		Size:         fileInfo.Size(),
	}

	return fileMeta, nil
}

// FileMetaWithCertificate returns a FileMeta struct with certificate metadata if applicable.
func FileMetaWithCertificate(filePath string) (*mpi.FileMeta, error) {
	fileMeta, err := FileMeta(filePath)
	if err != nil {
		return nil, err
	}

	loadedCert, certErr := cert.LoadCertificate(filePath)
	if certErr != nil {
		// If it's not a certificate, just return the base file metadata.
		return fileMeta, certErr
	}

	// Populate certificate-specific metadata
	fileMeta.FileType = &mpi.FileMeta_CertificateMeta{
		CertificateMeta: &mpi.CertificateMeta{
			SerialNumber: loadedCert.SerialNumber.String(),
			Issuer: &mpi.X509Name{
				Country:            loadedCert.Issuer.Country,
				Organization:       loadedCert.Issuer.Organization,
				OrganizationalUnit: loadedCert.Issuer.OrganizationalUnit,
				Locality:           loadedCert.Issuer.Locality,
				Province:           loadedCert.Issuer.Province,
				StreetAddress:      loadedCert.Issuer.StreetAddress,
				PostalCode:         loadedCert.Issuer.PostalCode,
				SerialNumber:       loadedCert.Issuer.SerialNumber,
				CommonName:         loadedCert.Issuer.CommonName,
			},
			Subject: &mpi.X509Name{
				Country:            loadedCert.Subject.Country,
				Organization:       loadedCert.Subject.Organization,
				OrganizationalUnit: loadedCert.Subject.OrganizationalUnit,
				Locality:           loadedCert.Subject.Locality,
				Province:           loadedCert.Subject.Province,
				StreetAddress:      loadedCert.Subject.StreetAddress,
				PostalCode:         loadedCert.Subject.PostalCode,
				SerialNumber:       loadedCert.Subject.SerialNumber,
				CommonName:         loadedCert.Subject.CommonName,
			},
			Sans: &mpi.SubjectAlternativeNames{
				DnsNames:    loadedCert.DNSNames,
				IpAddresses: convertIPToString(loadedCert.IPAddresses),
			},
			Dates: &mpi.CertificateDates{
				NotBefore: loadedCert.NotBefore.Unix(),
				NotAfter:  loadedCert.NotAfter.Unix(),
			},
			SignatureAlgorithm: convertX509SignatureAlgorithm(loadedCert.SignatureAlgorithm),
			PublicKeyAlgorithm: loadedCert.PublicKeyAlgorithm.String(),
		},
	}

	return fileMeta, nil
}

// Permissions returns a file's permissions as a string.
func Permissions(fileMode os.FileMode) string {
	return fmt.Sprintf("%#o", fileMode.Perm())
}

func FileMode(mode string) os.FileMode {
	result, err := strconv.ParseInt(mode, 8, 32)
	if err != nil {
		return os.FileMode(permissions)
	}

	return os.FileMode(result)
}

// GenerateConfigVersion returns a unique config version for a set of files.
// The config version is calculated by joining the file hashes together and generating a unique ID.
func GenerateConfigVersion(fileSlice []*mpi.File) string {
	var hashes string

	files := make([]*mpi.File, len(fileSlice))
	copy(files, fileSlice)
	slices.SortFunc(files, func(a, b *mpi.File) int {
		return cmp.Compare(a.GetFileMeta().GetName(), b.GetFileMeta().GetName())
	})

	for _, file := range files {
		hashes += file.GetFileMeta().GetHash()
	}

	return GenerateHash([]byte(hashes))
}

// GenerateHash returns the hash value of a file's contents.
func GenerateHash(b []byte) string {
	hash := sha256.New()
	hash.Write(b)

	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// ConvertToMapOfFiles converts a list of files to a map of files with the file name as the key
func ConvertToMapOfFiles(files []*mpi.File) map[string]*mpi.File {
	filesMap := make(map[string]*mpi.File)
	for _, file := range files {
		filesMap[file.GetFileMeta().GetName()] = file
	}

	return filesMap
}

func convertIPToString(ips []net.IP) []string {
	ipArray := make([]string, len(ips))

	for i, ip := range ips {
		ipArray[i] = ip.String()
	}

	return ipArray
}

func convertX509SignatureAlgorithm(alg x509.SignatureAlgorithm) mpi.SignatureAlgorithm {
	x509ToMpiSignatureMap := map[x509.SignatureAlgorithm]mpi.SignatureAlgorithm{
		x509.MD2WithRSA:                mpi.SignatureAlgorithm_MD2_WITH_RSA,
		x509.MD5WithRSA:                mpi.SignatureAlgorithm_MD5_WITH_RSA,
		x509.SHA1WithRSA:               mpi.SignatureAlgorithm_SHA1_WITH_RSA,
		x509.SHA256WithRSA:             mpi.SignatureAlgorithm_SHA256_WITH_RSA,
		x509.SHA384WithRSA:             mpi.SignatureAlgorithm_SHA384_WITH_RSA,
		x509.SHA512WithRSA:             mpi.SignatureAlgorithm_SHA512_WITH_RSA,
		x509.DSAWithSHA1:               mpi.SignatureAlgorithm_DSA_WITH_SHA1,
		x509.DSAWithSHA256:             mpi.SignatureAlgorithm_DSA_WITH_SHA256,
		x509.ECDSAWithSHA1:             mpi.SignatureAlgorithm_ECDSA_WITH_SHA1,
		x509.ECDSAWithSHA256:           mpi.SignatureAlgorithm_ECDSA_WITH_SHA256,
		x509.ECDSAWithSHA384:           mpi.SignatureAlgorithm_ECDSA_WITH_SHA384,
		x509.ECDSAWithSHA512:           mpi.SignatureAlgorithm_ECDSA_WITH_SHA512,
		x509.SHA256WithRSAPSS:          mpi.SignatureAlgorithm_SHA256_WITH_RSA_PSS,
		x509.SHA384WithRSAPSS:          mpi.SignatureAlgorithm_SHA384_WITH_RSA_PSS,
		x509.SHA512WithRSAPSS:          mpi.SignatureAlgorithm_SHA512_WITH_RSA_PSS,
		x509.PureEd25519:               mpi.SignatureAlgorithm_PURE_ED25519,
		x509.UnknownSignatureAlgorithm: mpi.SignatureAlgorithm_SIGNATURE_ALGORITHM_UNKNOWN,
	}
	if mappedAlg, exists := x509ToMpiSignatureMap[alg]; exists {
		return mappedAlg
	}

	return mpi.SignatureAlgorithm_SIGNATURE_ALGORITHM_UNKNOWN
}
