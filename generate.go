package main

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/square/certstrap/pkix"
)

func init() {
	rand.Seed(time.Now().Unix())
}

const (
	CfgNone             = 0
	CfgWithSubdomainURI = 1 << iota
	CfgWithHTTPSURL
)

type OutputData map[string]interface{}

func (o OutputData) AddSystemComponent(name string, cfgFlags int) {
	sysDomain := o["system_domain"]
	uri := fmt.Sprintf("%s.%s", name, sysDomain)
	o[fmt.Sprintf("%s_uri", name)] = uri

	if cfgFlags&CfgWithSubdomainURI != 0 {
		o[fmt.Sprintf("%s_subdomain_uri", name)] = fmt.Sprintf("*.%s", uri)
	}
	if cfgFlags&CfgWithHTTPSURL != 0 {
		o[fmt.Sprintf("%s_url", name)] = fmt.Sprintf("https://%s", uri)
	}
}

func generatePassword() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic("unable to read rand bytes: " + err.Error())
	}
	return strings.Trim(base64.RawURLEncoding.EncodeToString(bytes), "-_")
}

func (o OutputData) GeneratePasswords(keynames ...string) {
	for _, name := range keynames {
		o[name] = generatePassword()
	}
}

func (o OutputData) GeneratePasswordArray(keyName string, numKeys int) {
	var passwords []string
	for i := 0; i < numKeys; i++ {
		passwords = append(passwords, generatePassword())
	}
	o[keyName] = passwords
}

func (o OutputData) GeneratePlainKeyPair(plainKeyPair *PlainKeyPair) error {
	keyPair, err := pkix.CreateRSAKey(KeyBits)
	if err != nil {
		return fmt.Errorf("create key pair: %s", err)
	}

	private := keyPair.Private.(*rsa.PrivateKey)
	public := private.Public().(*rsa.PublicKey)

	o[plainKeyPair.VarName_PublicKey], err = PublicKeyToPEM(public)
	if err != nil {
		return fmt.Errorf("export public key pem: %s", err)
	}
	o[plainKeyPair.VarName_PrivateKey] = PrivateKeyToPEM(private)
	return nil
}

func (o OutputData) GenerateCerts(ca *CA, certKeyPairs ...*CertKeyPair) error {
	var err error
	if err = ca.Init(); err != nil {
		return fmt.Errorf("init ca: %s", err)
	}

	if len(ca.VarName_CA) > 0 {
		o[ca.VarName_CA], err = ca.CACertAsString()
		if err != nil {
			return err
		}
	}

	for _, certKeyPair := range certKeyPairs {
		err = ca.InitCertKeyPair(certKeyPair)
		if err != nil {
			return err
		}
		o[certKeyPair.VarName_Cert], err = certKeyPair.CertAsString()
		if err != nil {
			return err
		}
		o[certKeyPair.VarName_Key], err = certKeyPair.PrivateKeyAsString()
		if err != nil {
			return err
		}
	}

	return nil
}

type CertSet struct {
	CA           *CA
	CertKeyPairs []*CertKeyPair
}

func (cs *CertSet) Generate(o OutputData) error {
	return o.GenerateCerts(cs.CA, cs.CertKeyPairs...)
}
