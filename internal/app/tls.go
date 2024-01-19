package app

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"go.uber.org/zap"
	"math/big"
	"net"
	"os"
	"time"
)

const (
	serialNumber = 1
	ip4GrayZone  = 127
	yearsGrant   = 1
	RSALen       = 4096
	CertsPerm    = 0600
)

func CreateCertificates(logger *zap.SugaredLogger) error {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Organization: []string{"Shortener"},
			Country:      []string{"RU"},
		},
		IPAddresses:  []net.IP{net.IPv4(ip4GrayZone, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(yearsGrant, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, RSALen)
	if err != nil {
		logger.Fatal(err)
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		logger.Fatal(err)
	}
	certFile, err := os.OpenFile("./certs/cert.pem", os.O_WRONLY|os.O_CREATE, CertsPerm)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		if err := certFile.Close(); err != nil {
			logger.Fatal(err)
		}
	}()
	if err := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return fmt.Errorf("error creating cert file: %w", err)
	}
	rsaFile, err := os.OpenFile("./certs/private.pem", os.O_WRONLY|os.O_CREATE, CertsPerm)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		if err := rsaFile.Close(); err != nil {
			logger.Fatal(err)
		}
	}()
	if err := pem.Encode(rsaFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return fmt.Errorf("error creating RSA private key: %w", err)
	}
	return nil
}
