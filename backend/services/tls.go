package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"goWFM/config"
	"goWFM/db"
)

// EnsureTLSCert 确保 TLS 证书可用
// 如果提供了证书和私钥则解析使用，否则自动生成自签名证书并保存到数据库
func EnsureTLSCert(certPEM, keyPEM string) (tls.Certificate, error) {
	if certPEM != "" && keyPEM != "" {
		cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("parse provided TLS cert: %w", err)
		}
		return cert, nil
	}

	// 生成自签名证书
	certStr, keyStr, err := GenerateSelfSignedCert()
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate self-signed cert: %w", err)
	}

	// 保存到数据库
	appCfg := config.GetAppearance()
	appCfg.SSLCert = certStr
	appCfg.SSLKey = keyStr
	data, _ := json.Marshal(appCfg)
	db.DB.Exec(
		`INSERT INTO gowfm_config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		config.KeyAppearance, string(data),
	)
	config.SetAppearance(appCfg)

	cert, err := tls.X509KeyPair([]byte(certStr), []byte(keyStr))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse generated TLS cert: %w", err)
	}
	return cert, nil
}

// GenerateSelfSignedCert 生成 ECDSA P-256 自签名证书，有效期 1 年
func GenerateSelfSignedCert() (certPEM, keyPEM string, err error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"goWFM Self-Signed"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("create certificate: %w", err)
	}

	certBlock := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}
	keyBlock := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return string(certBlock), string(keyBlock), nil
}
