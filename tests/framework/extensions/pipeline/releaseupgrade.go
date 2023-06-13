package pipeline

import (
	"os"

	"github.com/rancher/rancher/tests/v2/validation/upgrade"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ReleaseUpgradeConfigKey is the key name of ReleaseUpgradeConfig values in the cattle config
const ReleaseUpgradeConfigKey = "releaseUpgrade"

// ReleaseUpgradeConfig is a struct that contains:
//   - MainConfig, which is the embedded yaml fields, and built on top of it.
//   - HA and Clusters inputs.
//   - Provisioning and upgrade test cases and packages.
type ReleaseUpgradeConfig struct {
	//metada configs
	HAConfig HAConfig `yaml:"ha"`
	Clusters Clusters `yaml:"clusters"`

	//test case configs
	TestCases TestCases `yaml:"testCases"`
}

// TestCasesConfigKey is the key name of TestCases values in the cattle config
const TestCasesConfigKey = "testCases"

// TestCases is a struct that contains related information about the required package and run test strings to run go test commands.
type TestCases struct {
	//provisioning test cases
	ProvisioningTestPackage string `yaml:"provisioningTestPackage"`
	ProvisioningTestCase    string `yaml:"provisioningTestCase"`

	//upgrade test cases
	UpgradeTestPackage        string `yaml:"upgradeTestCase"`
	UpgradeKubernetesTestCase string `yaml:"upgradeKubernetesTestCase"`
	PreUpgradeTestCase        string `yaml:"preUpgradeTestCase"`
	PostUpgradeTestCase       string `yaml:"postUpgradeTestCase"`
}

// HAConfigKey is the key name of HAConfig values in the cattle config
const HAConfigKey = "ha"

// HAConfig is a struct that contains related information about the HA that's going to be created and upgraded.
type HAConfig struct {
	Host                  string `yaml:"host"`
	ChartVersion          string `yaml:"chartVersion"`
	ChartVersionToUpgrade string `yaml:"chartVersionToUpgrade"`
	ImageTag              string `yaml:"imageTag"`
	ImageTagToUpgrade     string `yaml:"imageTagToUpgrade"`
	CertOption            string `yaml:"certOption"`
	Insecure              *bool  `yaml:"insecure" default:"true"`
	Cleanup               *bool  `yaml:"cleanup" default:"true"`
}

// ClustersConfigKey is the key name of Clusters values in the cattle config
const ClustersConfigKey = "clusters"

// Clusters is a struct that contains cluster types.
type Clusters struct {
	RKE1Clusters   RancherClusters `yaml:"rke1"`
	RKE2Clusters   RancherClusters `yaml:"rke2"`
	K3sClusters    RancherClusters `yaml:"k3s"`
	HostedClusters []HostedCluster `yaml:"hosted"`
}

// RancherClusters is a struct that contains slice of custom and node providers as ProviderCluster type.
type RancherClusters struct {
	CustomClusters       []RancherCluster `yaml:"custom"`
	NodeProviderClusters []RancherCluster `yaml:"nodeProvider"`
}

// RancherCluster is a struct that contains related information about the downstream cluster that's going to be created and upgraded.
type RancherCluster struct {
	Provider                   string           `yaml:"provider"`
	KubernetesVersion          string           `yaml:"kubernetesVersion"`
	KubernetesVersionToUpgrade string           `yaml:"kubernetesVersionToUpgrade"`
	Image                      string           `yaml:"image"`
	CNIs                       []string         `yaml:"cni"`
	FeaturesToTest             upgrade.Features `yaml:"enabledFeatures" default:""`
	SSHUser                    string           `yaml:"sshUser" default:""`
	VolumeType                 string           `yaml:"volumeType" default:""`
}

// HostedCluster is a struct that contains related information about the downstream cluster that's going to be created and upgraded.
type HostedCluster struct {
	Provider                   string           `yaml:"provider"`
	KubernetesVersion          string           `yaml:"kubernetesVersion"`
	KubernetesVersionToUpgrade string           `yaml:"kubernetesVersionToUpgrade"`
	FeaturesToTest             upgrade.Features `yaml:"enabledFeatures" default:""`
}

// GenerateDefaultReleaseUpgradeConfig is a function that creates the ReleaseUpgradeConfig with its default values.
func GenerateDefaultReleaseUpgradeConfig() {
	configFileName := "default-release-upgrade.yaml"

	config := new(ReleaseUpgradeConfig)

	configData, err := yaml.Marshal(&config)
	if err != nil {
		logrus.Fatalf("error marshaling: %v", err)
	}
	err = os.WriteFile(configFileName, configData, 0644)
	if err != nil {
		logrus.Fatalf("error writing yaml: %v", err)
	}
}
