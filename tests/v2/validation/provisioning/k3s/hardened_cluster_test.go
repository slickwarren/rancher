//go:build (validation || sanity) && !infra.any && !infra.aks && !infra.eks && !infra.rke2k3s && !infra.gke && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !extended && !stress

package k3s

import (
	"testing"

	cis "github.com/rancher/rancher/tests/v2/validation/provisioning/resources/cisbenchmark"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/charts"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/extensions/projects"
	"github.com/rancher/shepherd/extensions/provisioning"
	"github.com/rancher/shepherd/extensions/provisioninginput"
	"github.com/rancher/shepherd/extensions/users"
	password "github.com/rancher/shepherd/extensions/users/passwordgenerator"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HardenedK3SClusterProvisioningTestSuite struct {
	suite.Suite
	client              rancher.Client
	session             *session.Session
	standardUserClient  rancher.Client
	provisioningConfig  *provisioninginput.Config
	project             *management.Project
	chartInstallOptions *charts.InstallOptions
	chartFeatureOptions *charts.RancherMonitoringOpts
}

func (c *HardenedK3SClusterProvisioningTestSuite) TearDownSuite() {
	c.session.Cleanup()
}

func (c *HardenedK3SClusterProvisioningTestSuite) SetupSuite() {
	testSession := session.NewSession()
	c.session = testSession

	c.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, c.provisioningConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(c.T(), err)

	c.client = *client

	c.provisioningConfig.K3SKubernetesVersions, err = kubernetesversions.Default(&c.client, clusters.K3SClusterType.String(), c.provisioningConfig.K3SKubernetesVersions)
	require.NoError(c.T(), err)

	enabled := true
	var testuser = namegen.AppendRandomString("testuser-")
	var testpassword = password.GenerateUserPassword("testpass-")
	user := &management.User{
		Username: testuser,
		Password: testpassword,
		Name:     testuser,
		Enabled:  &enabled,
	}

	newUser, err := users.CreateUserWithRole(client, user, "user")
	require.NoError(c.T(), err)

	newUser.Password = user.Password

	standardUserClient, err := client.AsUser(newUser)
	require.NoError(c.T(), err)

	standardUserClient.Session.CleanupEnabled = false
	c.standardUserClient = *standardUserClient
}

func (c *HardenedK3SClusterProvisioningTestSuite) TestProvisioningK3SHardenedCluster() {
	c.T().Parallel()

	nodeRolesDedicated := []provisioninginput.MachinePools{provisioninginput.EtcdMachinePool, provisioninginput.ControlPlaneMachinePool, provisioninginput.WorkerMachinePool}

	tests := []struct {
		name            string
		client          *rancher.Client
		machinePools    []provisioninginput.MachinePools
		scanProfileName string
	}{
		{"K3S CIS 1.8 Profile Hardened " + provisioninginput.StandardClientName.String(), &c.standardUserClient, nodeRolesDedicated, "k3s-cis-1.8-profile-hardened"},
		{"K3S CIS 1.8 Profile Permissive " + provisioninginput.StandardClientName.String(), &c.standardUserClient, nodeRolesDedicated, "k3s-cis-1.8-profile-permissive"},
	}
	for _, tt := range tests {

		tt := tt
		c.Suite.T().Run(tt.name, func(t *testing.T) {
			provisioningConfig := *c.provisioningConfig
			provisioningConfig.MachinePools = tt.machinePools
			provisioningConfig.Hardened = true

			nodeProviders := provisioningConfig.NodeProviders[0]
			externalNodeProvider := provisioning.ExternalNodeProviderSetup(nodeProviders)

			testConfig := clusters.ConvertConfigToClusterConfig(&provisioningConfig)
			testConfig.KubernetesVersion = c.provisioningConfig.K3SKubernetesVersions[0]

			clusterObject, err := provisioning.CreateProvisioningCustomCluster(tt.client, &externalNodeProvider, testConfig)
			require.NoError(c.T(), err)

			provisioning.VerifyCluster(c.T(), tt.client, testConfig, clusterObject)

			cluster, err := clusters.NewClusterMeta(tt.client, clusterObject.Name)
			require.NoError(c.T(), err)

			latestCISBenchmarkVersion, err := tt.client.Catalog.GetLatestChartVersion(charts.CISBenchmarkName, catalog.RancherChartRepo)
			require.NoError(c.T(), err)

			project, err := projects.GetProjectByName(tt.client, cluster.ID, cis.System)
			require.NoError(c.T(), err)

			c.project = project
			require.NotEmpty(c.T(), c.project)

			c.chartInstallOptions = &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestCISBenchmarkVersion,
				ProjectID: c.project.ID,
			}

			cis.SetupCISBenchmarkChart(tt.client, c.project.ClusterID, c.chartInstallOptions, charts.CISBenchmarkNamespace)
			cis.RunCISScan(tt.client, c.project.ClusterID, tt.scanProfileName)
		})
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestHardenedK3SClusterProvisioningTestSuite(t *testing.T) {
	suite.Run(t, new(HardenedK3SClusterProvisioningTestSuite))
}
