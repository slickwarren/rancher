//go:build validation

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
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RKE2PSACTTestSuite struct {
	suite.Suite
	client             rancher.Client
	session            *session.Session
	standardUserClient rancher.Client
	provisioningConfig *provisioninginput.Config
}

func (r *RKE2PSACTTestSuite) TearDownSuite() {
	r.session.Cleanup()
}

func (r *RKE2PSACTTestSuite) SetupSuite() {
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

func (r *RKE2PSACTTestSuite) TestRKE2PSACTNodeDriverCluster() {
	r.T().Parallel()

	nodeRolesDedicated := []provisioninginput.MachinePools{
		provisioninginput.EtcdMachinePool,
		provisioninginput.ControlPlaneMachinePool,
		provisioninginput.WorkerMachinePool,
	}

	tests := []struct {
		name         string
		machinePools []provisioninginput.MachinePools
		psact        provisioninginput.PSACT
		client       rancher.Client
	}{
		{
			name:         "Rancher Privileged " + provisioninginput.StandardClientName.String(),
			machinePools: nodeRolesDedicated,
			psact:        "rancher-privileged",
			client:       r.standardUserClient,
		},
		{
			name:         "Rancher Restricted " + provisioninginput.StandardClientName.String(),
			machinePools: nodeRolesDedicated,
			psact:        "rancher-restricted",
			client:       r.standardUserClient,
		},
		{
			name:         "Rancher Baseline " + provisioninginput.AdminClientName.String(),
			machinePools: nodeRolesDedicated,
			psact:        "rancher-baseline",
			client:       r.client,
		},
	}

	for _, tt := range tests {
		tt := tt
		provisioningConfig := *r.provisioningConfig
		provisioningConfig.MachinePools = tt.machinePools
		provisioningConfig.PSACT = string(tt.psact)
		r.Suite.T().Run(tt.name, func(t *testing.T) {
			permutations.RunTestPermutations(&r.Suite, tt.name, &tt.client, &provisioningConfig,
				permutations.RKE2ProvisionCluster, nil, nil)
		})
	}
}

func (r *RKE2PSACTTestSuite) TestRKE2PSACTCustomCluster() {
	r.T().Parallel()

	nodeRolesDedicated := []provisioninginput.MachinePools{
		provisioninginput.EtcdMachinePool,
		provisioninginput.ControlPlaneMachinePool,
		provisioninginput.WorkerMachinePool,
	}

	tests := []struct {
		name         string
		machinePools []provisioninginput.MachinePools
		psact        provisioninginput.PSACT
		client       rancher.Client
	}{
		{
			name:         "Rancher Privileged " + provisioninginput.StandardClientName.String(),
			machinePools: nodeRolesDedicated,
			psact:        "rancher-privileged",
			client:       r.standardUserClient,
		},
		{
			name:         "Rancher Restricted " + provisioninginput.StandardClientName.String(),
			machinePools: nodeRolesDedicated,
			psact:        "rancher-restricted",
			client:       r.standardUserClient,
		},
		{
			name:         "Rancher Baseline " + provisioninginput.AdminClientName.String(),
			machinePools: nodeRolesDedicated,
			psact:        "rancher-baseline",
			client:       r.client,
		},
	}

	for _, tt := range tests {
		tt := tt
		provisioningConfig := *r.provisioningConfig
		provisioningConfig.MachinePools = tt.machinePools
		provisioningConfig.PSACT = string(tt.psact)
		r.Suite.T().Run(tt.name, func(t *testing.T) {
			permutations.RunTestPermutations(&r.Suite, tt.name, &tt.client, &provisioningConfig,
				permutations.RKE2CustomCluster, nil, nil)
		})
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestRKE2PSACTTestSuite(t *testing.T) {
	suite.Run(t, new(RKE2PSACTTestSuite))
}
