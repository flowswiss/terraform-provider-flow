package flow

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccComputeCertificate_Basic(t *testing.T) {
	commonName := "flow.swiss"
	orgName := "Flow Swiss AG"

	certificateName := acctest.RandomWithPrefix("test-certificate")
	cert, priv, err := randTLSCert(commonName, orgName)
	if err != nil {
		t.Fatal(err)
	}

	certBase64 := base64.StdEncoding.EncodeToString([]byte(cert))
	privBase64 := base64.StdEncoding.EncodeToString([]byte(priv))

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeCertificateConfigBasic, certificateName, certBase64, privBase64),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_certificate.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_certificate.foobar", "name", certificateName),
					resource.TestCheckResourceAttr("flow_compute_certificate.foobar", "location_id", "1"),
					resource.TestCheckNoResourceAttr("flow_compute_certificate.foobar", "certificate"),
					resource.TestCheckNoResourceAttr("flow_compute_certificate.foobar", "private_key"),
					resource.TestCheckResourceAttrSet("flow_compute_certificate.foobar", "info.not_before"),
					resource.TestCheckResourceAttrSet("flow_compute_certificate.foobar", "info.not_after"),
					resource.TestCheckResourceAttrSet("flow_compute_certificate.foobar", "info.serial_number"),
				),
			},
			{
				ResourceName:      "flow_compute_certificate.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:    "flow_compute_certificate.foobar",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
			},
			{
				Config: fmt.Sprintf(testAccComputeCertificateConfigBasic, certificateName, certBase64, privBase64) + testAccComputeCertificateConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "id", "flow_compute_certificate.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "name", "flow_compute_certificate.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "location_id", "flow_compute_certificate.foobar", "location_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.subject.common_name", "flow_compute_certificate.foobar", "info.subject.common_name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.subject.organizational_unit", "flow_compute_certificate.foobar", "info.subject.organizational_unit"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.subject.organization", "flow_compute_certificate.foobar", "info.subject.organization"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.subject.locality", "flow_compute_certificate.foobar", "info.subject.locality"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.subject.province", "flow_compute_certificate.foobar", "info.subject.province"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.subject.country", "flow_compute_certificate.foobar", "info.subject.country"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.issuer.common_name", "flow_compute_certificate.foobar", "info.issuer.common_name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.issuer.organizational_unit", "flow_compute_certificate.foobar", "info.issuer.organizational_unit"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.issuer.organization", "flow_compute_certificate.foobar", "info.issuer.organization"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.issuer.locality", "flow_compute_certificate.foobar", "info.issuer.locality"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.issuer.province", "flow_compute_certificate.foobar", "info.issuer.province"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.issuer.country", "flow_compute_certificate.foobar", "info.issuer.country"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.not_before", "flow_compute_certificate.foobar", "info.not_before"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.not_after", "flow_compute_certificate.foobar", "info.not_after"),
					resource.TestCheckResourceAttrPair("data.flow_compute_certificate.by_name", "info.serial_number", "flow_compute_certificate.foobar", "info.serial_number"),
				),
			},
		},
	})
}

const testAccComputeCertificateConfigBasic = `
resource "flow_compute_certificate" "foobar" {
	name        = "%s"
	location_id = 1

	certificate = "%s"
	private_key = "%s"
}
`

// taken from https://github.com/hashicorp/terraform-plugin-sdk/blob/70ce77bce6118b74a49762bb401b46a723c0bab8/helper/acctest/random.go#L77
func randTLSCert(commonName string, orgName string) (string, string, error) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(int64(acctest.RandInt())),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{orgName},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	privateKey, privateKeyPEM, err := genPrivateKey()
	if err != nil {
		return "", "", err
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	certPEM, err := pemEncode(cert, "CERTIFICATE")
	if err != nil {
		return "", "", err
	}

	return certPEM, privateKeyPEM, nil
}

func genPrivateKey() (*rsa.PrivateKey, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, "", err
	}

	privateKeyPEM, err := pemEncode(x509.MarshalPKCS1PrivateKey(privateKey), "RSA PRIVATE KEY")
	if err != nil {
		return nil, "", err
	}

	return privateKey, privateKeyPEM, nil
}

func pemEncode(b []byte, block string) (string, error) {
	var buf bytes.Buffer
	pb := &pem.Block{Type: block, Bytes: b}
	if err := pem.Encode(&buf, pb); err != nil {
		return "", err
	}

	return buf.String(), nil
}

const testAccComputeCertificateConfigDataSource = `
data "flow_compute_certificate" "by_name" {
	name        = flow_compute_certificate.foobar.name
	location_id = flow_compute_certificate.foobar.location_id
}
`
