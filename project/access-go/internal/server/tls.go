package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"os"
	"time"
)

// generateSelfSignedTLSConfig 生成或加载自签名证书（仅用于开发环境）
func generateSelfSignedTLSConfig() (*tls.Config, error) {
	certFile := "dev_cert.pem"
	keyFile := "dev_key.pem"

	// 1. 尝试加载现有证书
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err == nil {
		slog.Info("Loaded existing dev certificate", "cert", certFile)
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h3", "webtransport"},
			MinVersion:   tls.VersionTLS13,
		}, nil
	}

	// 2. 生成新证书
	slog.Info("Generating new dev certificate...")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"IM Dev"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),      // Backdate slightly to avoid clock skew
		NotAfter:              time.Now().Add(24 * time.Hour * 10), // Max 14 days for WebTransport self-signed
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	// 3. 保存到文件
	certOut, err := os.Create(certFile)
	if err != nil {
		return nil, err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return nil, err
	}

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	defer keyOut.Close()
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, err
	}

	slog.Info("Device certificate saved", "cert", certFile, "key", keyFile)

	// 4. 返回配置
	cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3", "webtransport"},
		MinVersion:   tls.VersionTLS13,
	}, nil
}
