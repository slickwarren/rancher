//go:build (validation || extended) && !infra.any && !infra.aks && !infra.eks && !infra.gke && !infra.rke2k3s && !cluster.any && !cluster.custom && !cluster.nodedriver && !sanity && !stress

package rke2

import (
	"testing"

	"github.com/rancher/rancher/tests/v2/validation/provisioning/permutations"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/extensions/provisioninginput"
	"github.com/rancher/shepherd/extensions/users"
	password "github.com/rancher/shepherd/extensions/users/passwordgenerator"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/environmentflag"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RKE2NodeDriverProvisioningTestSuite struct {
	suite.Suite
	client             rancher.Client
	session            *session.Session
	standardUserClient rancher.Client
	provisioningConfig *provisioninginput.Config
}

func (r *RKE2NodeDriverProvisioningTestSuite) TearDownSuite() {
	logrus.Info("Cleaning up now...")
	r.session.Cleanup()
}

func (r *RKE2NodeDriverProvisioningTestSuite) SetupSuite() {
	testSession := session.NewSession()
	r.session = testSession

	r.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, r.provisioningConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(r.T(), err)
	r.client = *client

	r.provisioningConfig.RKE2KubernetesVersions, err = kubernetesversions.Default(
		&r.client, clusters.RKE2ClusterType.String(), r.provisioningConfig.RKE2KubernetesVersions)
	require.NoError(r.T(), err)

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
	require.NoError(r.T(), err)

	newUser.Password = user.Password

	standardUserClient, err := client.AsUser(newUser)
	require.NoError(r.T(), err)
	standardUserClient.Session.CleanupEnabled = false

	r.standardUserClient = *standardUserClient
}

func (r *RKE2NodeDriverProvisioningTestSuite) TestProvisioningRKE2Cluster() {
	r.T().Parallel()

	nodeRolesAll := []provisioninginput.MachinePools{provisioninginput.AllRolesMachinePool}
	nodeRolesShared := []provisioninginput.MachinePools{provisioninginput.EtcdControlPlaneMachinePool, provisioninginput.WorkerMachinePool}
	nodeRolesDedicated := []provisioninginput.MachinePools{provisioninginput.EtcdMachinePool, provisioninginput.ControlPlaneMachinePool, provisioninginput.WorkerMachinePool}

	tests := []struct {
		name         string
		machinePools []provisioninginput.MachinePools
		client       rancher.Client
		runFlag      bool
	}{
		{"1 Node all roles " + provisioninginput.StandardClientName.String(), nodeRolesAll, r.standardUserClient, r.client.Flags.GetValue(environmentflag.Short) || r.client.Flags.GetValue(environmentflag.Long)},
		{"2 nodes - etcd|cp roles per 1 node " + provisioninginput.StandardClientName.String(), nodeRolesShared, r.standardUserClient, r.client.Flags.GetValue(environmentflag.Short) || r.client.Flags.GetValue(environmentflag.Long)},
		{"3 nodes - 1 role per node " + provisioninginput.StandardClientName.String(), nodeRolesDedicated, r.standardUserClient, r.client.Flags.GetValue(environmentflag.Long)},
	}

	for _, tt := range tests {
		if !tt.runFlag {
			r.T().Logf("SKIPPED")
			continue
		}

		tt := tt
		r.Suite.T().Run(tt.name, func(t *testing.T) {
			provisioningConfig := *r.provisioningConfig
			provisioningConfig.MachinePools = tt.machinePools
			permutations.RunTestPermutations(&r.Suite, tt.name, &tt.client, &provisioningConfig, permutations.RKE2ProvisionCluster, nil, nil)
		})
	}
	if *r.client.RancherConfig.Cleanup {
		r.standardUserClient.Session.CleanupEnabled = true
		r.Suite.T().Cleanup(r.standardUserClient.Session.Cleanup)
	}
}

func (r *RKE2NodeDriverProvisioningTestSuite) TestProvisioningRKE2ClusterDynamicInput() {
	r.T().Parallel()

	if len(r.provisioningConfig.MachinePools) == 0 {
		r.T().Skip()
	}

	tests := []struct {
		name   string
		client rancher.Client
	}{
		{provisioninginput.StandardClientName.String(), r.standardUserClient},
	}
	for _, tt := range tests {
		tt := tt

		r.Suite.T().Run(tt.name, func(t *testing.T) {
			permutations.RunTestPermutations(&r.Suite, tt.name, &tt.client, r.provisioningConfig, permutations.RKE2ProvisionCluster, nil, nil)
		})
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestRKE2ProvisioningTestSuite(t *testing.T) {
	suite.Run(t, new(RKE2NodeDriverProvisioningTestSuite))
}
