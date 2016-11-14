package main

import (
	"fmt"

	"github.com/rosenhouse/cf-filler/creds"
	"github.com/rosenhouse/cf-filler/vars"
)

const (
	CfgNone             = 0
	CfgWithSubdomainURI = 1 << iota
	CfgWithHTTPSURL
)

type DeploymentVars map[string]interface{}

func (o DeploymentVars) AddSystemComponent(name string, cfgFlags int) {
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

func (o DeploymentVars) GeneratePasswords(keynames ...string) {
	for _, name := range keynames {
		o[name] = creds.NewPassword()
	}
}

func (o DeploymentVars) GeneratePasswordArray(keyName string, numKeys int) {
	var passwords []string
	for i := 0; i < numKeys; i++ {
		passwords = append(passwords, creds.NewPassword())
	}
	o[keyName] = passwords
}

func (o DeploymentVars) GeneratePlainKeyPair(plainKeyPair *vars.PlainKeyPair) error {
	private, public, err := creds.NewRSAKeyPair()
	if err != nil {
		return fmt.Errorf("create RSA key pair: %s", err)
	}

	o[plainKeyPair.VarName_PublicKey] = public
	o[plainKeyPair.VarName_PrivateKey] = private
	return nil
}

func (o DeploymentVars) GenerateCerts(desiredCertSet *vars.CertSet) error {
	ca := creds.CA{
		CommonName: desiredCertSet.CA.CommonName,
	}
	certKeyPairs := make([]*creds.CertKeyPair, len(desiredCertSet.CertKeyPairs))
	for i, desiredCertKeyPair := range desiredCertSet.CertKeyPairs {
		certKeyPairs[i] = &creds.CertKeyPair{
			CommonName: desiredCertKeyPair.CommonName,
			Domains:    desiredCertKeyPair.Domains,
		}
	}

	var err error
	if err = ca.Init(); err != nil {
		return fmt.Errorf("init ca: %s", err)
	}

	if len(desiredCertSet.CA.VarName_CA) > 0 {
		o[desiredCertSet.CA.VarName_CA], err = ca.CACertAsString()
		if err != nil {
			return err
		}
	}

	for i, certKeyPair := range certKeyPairs {
		private, cert, err := ca.NewCertKeyPair(certKeyPair)
		if err != nil {
			return err
		}
		o[desiredCertSet.CertKeyPairs[i].VarName_Cert] = cert
		o[desiredCertSet.CertKeyPairs[i].VarName_Key] = private
	}

	return nil
}

func (o DeploymentVars) GenerateSSHKeyAndFingerprint(keyName string, fingerprintName string) error {
	sshPrivateKey, sshKeyFingerprint, err := creds.NewSSHKeyAndFingerprint()
	if err != nil {
		return fmt.Errorf("generate ssh key and fingerprint: %s", err)
	}

	o[keyName] = sshPrivateKey
	o[fingerprintName] = sshKeyFingerprint
	return nil
}
