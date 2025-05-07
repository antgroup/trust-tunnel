// Copyright The TrustTunnel Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

// TLSConfig defines the options for TLS configuration, including CA, certificate, and key.
// It is used to secure data transmission by configuring TLS connections.
type TLSConfig struct {
	// TLSVerify indicates whether to verify the server's certificate.
	TLSVerify bool `toml:"tls_verify"`
	// TLSCA is the path to the TLS Certificate Authority (CA) certificate.
	// It is used to verify the legitimacy of server and client certificates.
	TLSCA string `toml:"tls_ca"`
	// TLSCert is the path to the server's TLS certificate.
	// This certificate is used to prove the server's identity and encrypt data during transmission.
	TLSCert string `toml:"tls_cert"`
	// TLSKey is the path to the server's TLS private key.
	// Paired with TLSCert, it is used to decrypt received data and sign data being sent.
	TLSKey string `toml:"tls_key"`
}

// NTLSConfig is a structure used to configure Non-Traditional Layer Security (NTLS)
// It includes configurations related to certificate and key verification, signing, encryption, as well as cipher suite settings.
type NTLSConfig struct {
	// NTLSVerify indicates whether NTLS certificate verification is enabled.
	// When set to true, the remote server's NTLS certificate will be verified; when false, no verification occurs, which may pose security risks.
	NTLSVerify bool `toml:"ntls_verify"`

	// NTLSSignCertFile is the path to the NTLS certificate file used for signing.
	// This certificate proves the server's identity and its validity is checked by the client.
	NTLSSignCertFile string `toml:"ntls_sign_cert_file"`

	// NTLSSignKeyFile is the path to the key file for the NTLS certificate used for signing.
	// This key is paired with NTLSSignCertFile to decrypt and verify the certificate's signature.
	NTLSSignKeyFile string `toml:"ntls_sign_key_file"`

	// NTLSEncCertFile is the path to the NTLS certificate file used for encryption.
	// This certificate encrypts data to ensure confidentiality during transmission.
	NTLSEncCertFile string `toml:"ntls_enc_cert_file"`

	// NTLSEncKeyFile is the path to the key file for the NTLS certificate used for encryption.
	// This key is paired with NTLSEncCertFile to decrypt encrypted data.
	NTLSEncKeyFile string `toml:"ntls_enc_key_file"`

	// NTLSCaFile is the path to the certificate of the trusted NTLS Certificate Authority (CA).
	// The client uses this CA certificate to verify that the server's certificate was issued by a trusted authority.
	NTLSCaFile string `toml:"ntls_ca_file"`

	// Cipher defines the cipher suite used.
	// It specifies the encryption algorithm, key exchange, and authentication method.
	Cipher string `toml:"cipher"`
}

// The Server interface defines the method for starting a server.
// Any server should implement this interface to provide the capability of being started.
type Server interface {
	// Start starts the server with the provided options.
	Start(opt *Option) error
}
