package rbac

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	v1 "github.com/rancher/rancher/tests/framework/clients/rancher/v1"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/extensions/namespaces"
	"github.com/rancher/rancher/tests/framework/extensions/users"
	namegen "github.com/rancher/rancher/tests/framework/pkg/namegenerator"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	containerImage = "nginx"
	containerName  = "psa-nginx"
)

type PSATestSuite struct {
	suite.Suite
	client              *rancher.Client
	nonAdminUser        *management.User
	nonAdminUserClient  *rancher.Client
	session             *session.Session
	cluster             *management.Cluster
	adminProject        *management.Project
	stdUserProject      *management.Project
	steveAdminClient    *v1.Client
	steveNonAdminClient *v1.Client
	adminNamespace      *v1.SteveAPIObject
	stdUserNamespace    *v1.SteveAPIObject
	psaRole             *management.RoleTemplate
}

func (rb *PSATestSuite) TearDownSuite() {
	rb.session.Cleanup()
}

func (rb *PSATestSuite) SetupSuite() {
	testSession := session.NewSession()
	rb.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(rb.T(), err)

	rb.client = client
	clusterName := client.RancherConfig.ClusterName
	require.NotEmptyf(rb.T(), clusterName, "Cluster name to install should be set")
	clusterID, err := clusters.GetClusterIDByName(rb.client, clusterName)
	require.NoError(rb.T(), err, "Error getting cluster ID")
	rb.cluster, err = rb.client.Management.Cluster.ByID(clusterID)
	require.NoError(rb.T(), err)

	context := "cluster"
	roleName := namegen.AppendRandomString("psarole-")
	rules := []management.PolicyRule{
		{
			APIGroups: []string{"management.cattle.io"},
			Resources: []string{"projects"},
			Verbs:     []string{psaRole},
		},
	}
	psaRollUpdate, err := createRole(rb.client, context, roleName, rules)
	require.NoError(rb.T(), err)
	rb.psaRole = psaRollUpdate
}

func (rb *PSATestSuite) ValidatePSA(role string, customRole bool) {
	labels := map[string]string{
		psaWarn:    pssPrivilegedPolicy,
		psaEnforce: pssPrivilegedPolicy,
		psaAudit:   pssPrivilegedPolicy,
	}

	rb.T().Logf("Validate updating the PSA labels as %v", role)

	updateNS, err := getAndConvertNamespace(rb.adminNamespace, rb.steveAdminClient)
	require.NoError(rb.T(), err)
	updateNS.Labels = labels

	response, err := rb.steveNonAdminClient.SteveType(namespaces.NamespaceSteveType).Update(rb.adminNamespace, updateNS)

	switch role {
	case restrictedAdmin, roleOwner:
		require.NoError(rb.T(), err)
		expectedLabels := getPSALabels(response, labels)
		assert.Equal(rb.T(), labels, expectedLabels)
	case roleMember, roleProjectReadOnly:
		require.Error(rb.T(), err)
		errMessage := strings.Split(err.Error(), ":")[0]
		assert.Equal(rb.T(), "Resource type [namespace] is not updatable", errMessage)
	case roleProjectOwner, roleProjectMember, roleCustomCreateNS:
		require.Error(rb.T(), err)
		errStatus := strings.Split(err.Error(), ".")[1]
		rgx := regexp.MustCompile(`\[(.*?)\]`)
		errorMsg := rgx.FindStringSubmatch(errStatus)
		assert.Equal(rb.T(), "403 Forbidden", errorMsg[1])
	}

	rb.T().Logf("Validate deletion of the PSA labels as %v", role)

	deleteLabels(labels)

	deleteLabelsNS, err := getAndConvertNamespace(rb.adminNamespace, rb.steveAdminClient)
	require.NoError(rb.T(), err)
	deleteLabelsNS.Labels = labels

	_, err = rb.steveNonAdminClient.SteveType(namespaces.NamespaceSteveType).Update(rb.adminNamespace, deleteLabelsNS)
	switch role {
	case restrictedAdmin, roleOwner:
		require.NoError(rb.T(), err)
		expectedLabels := getPSALabels(response, labels)
		assert.Equal(rb.T(), 0, len(expectedLabels))
		if !customRole {
			_, err = createDeploymentAndWait(rb.steveNonAdminClient, rb.client, rb.cluster.ID, containerName, containerImage, rb.adminNamespace.Name)
			require.NoError(rb.T(), err)
		}
	case roleMember, roleProjectReadOnly:
		require.Error(rb.T(), err)
		errMessage := strings.Split(err.Error(), ":")[0]
		assert.Equal(rb.T(), "Resource type [namespace] is not updatable", errMessage)
	case roleProjectOwner, roleProjectMember, roleCustomCreateNS:
		errStatus := strings.Split(err.Error(), ".")[1]
		rgx := regexp.MustCompile(`\[(.*?)\]`)
		errorMsg := rgx.FindStringSubmatch(errStatus)
		assert.Equal(rb.T(), "403 Forbidden", errorMsg[1])
	}

	rb.T().Logf("Validate creation of new namespace with PSA labels as %v", role)

	labels = map[string]string{
		psaWarn:    pssBaselinePolicy,
		psaEnforce: pssBaselinePolicy,
		psaAudit:   pssBaselinePolicy,
	}
	namespaceName := namegen.AppendRandomString("testns-")
	namespaceCreate, err := namespaces.CreateNamespace(rb.nonAdminUserClient, namespaceName, "{}", labels, map[string]string{}, rb.adminProject)

	switch role {
	case restrictedAdmin, roleOwner:
		require.NoError(rb.T(), err)
		expectedLabels := getPSALabels(response, labels)
		assert.Equal(rb.T(), labels, expectedLabels)
		if !customRole {
			_, err = createDeploymentAndWait(rb.steveNonAdminClient, rb.client, rb.cluster.ID, containerName, containerImage, namespaceCreate.Name)
			require.NoError(rb.T(), err)
		}
	case roleProjectOwner, roleProjectMember, roleCustomCreateNS:
		require.Error(rb.T(), err)
		errStatus := strings.Split(err.Error(), ".")[1]
		rgx := regexp.MustCompile(`\[(.*?)\]`)
		errorMsg := rgx.FindStringSubmatch(errStatus)
		assert.Equal(rb.T(), "403 Forbidden", errorMsg[1])
	case roleMember, roleProjectReadOnly:
		require.Error(rb.T(), err)
		errMessage := strings.Split(err.Error(), ":")[0]
		assert.Equal(rb.T(), "Resource type [namespace] is not creatable", errMessage)
	}
}

func (rb *PSATestSuite) ValidateAdditionalPSA(role string) {
	createProjectAsNonAdmin, err := createProject(rb.nonAdminUserClient, rb.cluster.ID)
	require.NoError(rb.T(), err)
	rb.stdUserProject = createProjectAsNonAdmin

	relogin, err := rb.nonAdminUserClient.ReLogin()
	require.NoError(rb.T(), err)
	rb.nonAdminUserClient = relogin

	steveStdUserclient, err := rb.nonAdminUserClient.Steve.ProxyDownstream(rb.cluster.ID)
	require.NoError(rb.T(), err)
	rb.steveNonAdminClient = steveStdUserclient

	namespaceName := namegen.AppendRandomString("testns-")
	createNamespace, err := namespaces.CreateNamespace(rb.nonAdminUserClient, namespaceName, "{}",
		map[string]string{}, map[string]string{}, rb.stdUserProject)
	require.NoError(rb.T(), err)
	rb.stdUserNamespace = createNamespace

	rb.T().Logf("Validate editing new namespace in a cluster member created project with PSA labels as %v", role)
	labels := map[string]string{
		psaWarn:    pssRestrictedPolicy,
		psaEnforce: pssRestrictedPolicy,
		psaAudit:   pssRestrictedPolicy,
	}
	updateNS, err := getAndConvertNamespace(rb.stdUserNamespace, rb.steveAdminClient)
	require.NoError(rb.T(), err)
	updateNS.Labels = labels

	relogin, err = rb.nonAdminUserClient.ReLogin()
	require.NoError(rb.T(), err)
	rb.nonAdminUserClient = relogin

	steveStdUserclient, err = rb.nonAdminUserClient.Steve.ProxyDownstream(rb.cluster.ID)
	require.NoError(rb.T(), err)
	rb.steveNonAdminClient = steveStdUserclient

	response, err := rb.steveNonAdminClient.SteveType(namespaces.NamespaceSteveType).Update(rb.stdUserNamespace, updateNS)

	switch role {
	case roleOwner, psaRole:
		require.NoError(rb.T(), err)
		expectedLabels := getPSALabels(response, labels)
		assert.Equal(rb.T(), labels, expectedLabels)
		_, err = createDeploymentAndWait(rb.steveNonAdminClient, rb.client, rb.cluster.ID, containerName, containerImage, rb.stdUserNamespace.Name)
		require.Error(rb.T(), err)
	case roleMember:
		require.Error(rb.T(), err)
		errStatus := strings.Split(err.Error(), ".")[1]
		rgx := regexp.MustCompile(`\[(.*?)\]`)
		errorMsg := rgx.FindStringSubmatch(errStatus)
		assert.Equal(rb.T(), "403 Forbidden", errorMsg[1])
		updateNS, err := getAndConvertNamespace(rb.stdUserNamespace, rb.steveAdminClient)
		require.NoError(rb.T(), err)
		updateNS.Labels = labels
		_, err = rb.steveAdminClient.SteveType(namespaces.NamespaceSteveType).Update(rb.stdUserNamespace, updateNS)
		require.NoError(rb.T(), err)
	}

	rb.T().Logf("Validate deletion of PSA labels in namespace in a cluster member created project as %v", role)

	deleteLabels(labels)
	deleteLabelsNS, err := getAndConvertNamespace(rb.stdUserNamespace, rb.steveAdminClient)
	require.NoError(rb.T(), err)
	deleteLabelsNS.Labels = labels

	_, err = rb.steveNonAdminClient.SteveType(namespaces.NamespaceSteveType).Update(rb.stdUserNamespace, deleteLabelsNS)

	switch role {
	case roleOwner, psaRole:
		require.NoError(rb.T(), err)
		expectedLabels := getPSALabels(response, labels)
		assert.Equal(rb.T(), labels, expectedLabels)
		_, err = createDeploymentAndWait(rb.steveNonAdminClient, rb.client, rb.cluster.ID, containerName, containerImage, rb.stdUserNamespace.Name)
		require.NoError(rb.T(), err)
	case roleMember:
		require.Error(rb.T(), err)
		errStatus := strings.Split(err.Error(), ".")[1]
		rgx := regexp.MustCompile(`\[(.*?)\]`)
		errorMsg := rgx.FindStringSubmatch(errStatus)
		assert.Equal(rb.T(), "403 Forbidden", errorMsg[1])
	}
}

func (rb *PSATestSuite) TestPSA() {
	nonAdminUserRoles := [...]string{roleMember, restrictedAdmin, roleOwner, roleProjectOwner , roleProjectReadOnly, roleProjectMember, roleCustomCreateNS} 
	for _, role := range nonAdminUserRoles {
		var customRole bool
		if role == roleCustomCreateNS {
			customRole = true
		}
		rb.Run("Add PSA labels on the namespaces created by admins ", func() {
			createProjectAsAdmin, err := createProject(rb.client, rb.cluster.ID)
			rb.adminProject = createProjectAsAdmin
			require.NoError(rb.T(), err)

			steveAdminClient, err := rb.client.Steve.ProxyDownstream(rb.cluster.ID)
			require.NoError(rb.T(), err)
			rb.steveAdminClient = steveAdminClient
			namespaceName := namegen.AppendRandomString("testns-")
			labels := map[string]string{
				psaWarn:    pssRestrictedPolicy,
				psaEnforce: pssRestrictedPolicy,
				psaAudit:   pssRestrictedPolicy,
			}
			adminNamespace, err := namespaces.CreateNamespace(rb.client, namespaceName+"-admin", "{}", labels, map[string]string{}, rb.adminProject)
			require.NoError(rb.T(), err)
			expectedPSALabels := getPSALabels(adminNamespace, labels)
			assert.Equal(rb.T(), labels, expectedPSALabels)
			rb.adminNamespace = adminNamespace
			_, err = createDeploymentAndWait(rb.steveAdminClient, rb.client, rb.cluster.ID, containerName, containerImage, rb.adminNamespace.Name)
			require.Error(rb.T(), err)
		})

		rb.Run("Create a user with global role "+role, func() {
			var userRole string
			if role == restrictedAdmin {
				userRole = restrictedAdmin
			} else {
				userRole = standardUser
			}
			newUser, err := createUser(rb.client, userRole)
			require.NoError(rb.T(), err)
			rb.nonAdminUser = newUser
			rb.T().Logf("Created user: %v", rb.nonAdminUser.Username)
			rb.nonAdminUserClient, err = rb.client.AsUser(newUser)
			require.NoError(rb.T(), err)

			subSession := rb.session.NewSession()
			defer subSession.Cleanup()

			log.Info("Adding user as " + role + " to the downstream cluster.")
			if role != restrictedAdmin {
				if strings.Contains(role, "project") || role == roleProjectReadOnly || role == roleCustomCreateNS {
					err := users.AddProjectMember(rb.client, rb.adminProject, rb.nonAdminUser, role)
					require.NoError(rb.T(), err)
				} else {
					err := users.AddClusterRoleToUser(rb.client, rb.cluster, rb.nonAdminUser, role)
					require.NoError(rb.T(), err)
				}
				rb.nonAdminUserClient, err = rb.nonAdminUserClient.ReLogin()
				require.NoError(rb.T(), err)
			}

			steveClient, err := rb.nonAdminUserClient.Steve.ProxyDownstream(rb.cluster.ID)
			require.NoError(rb.T(), err)
			rb.steveNonAdminClient = steveClient
		})

		rb.Run("Testcase - Validate if members with roles "+role+"can add/edit/delete labesl from admin created namespace", func() {

			rb.ValidatePSA(role, customRole)
		})

		if strings.Contains(role, "cluster") {
			rb.Run("Additional testcase - Validate if members with roles "+role+"can add/edit/delete labels from admin created namespace", func() {
				rb.ValidateAdditionalPSA(role)
			})
		}

		if strings.Contains(role, "project") || role == roleCustomCreateNS {
			rb.Run("Additional testcase - Validate if "+role+" with an additional role update-psa can add/edit/delete labels from admin created namespace", func() {
				err := users.AddClusterRoleToUser(rb.client, rb.cluster, rb.nonAdminUser, rb.psaRole.ID)
				require.NoError(rb.T(), err)
				rb.ValidatePSA(psaRole, customRole)
			})
		}
	}
}

func TestPSATestSuite(t *testing.T) {
	suite.Run(t, new(PSATestSuite))
}
