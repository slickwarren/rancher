package planner

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/golang/mock/gomock"
	rkev1 "github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1"
	"github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1/plan"
	"github.com/rancher/rancher/pkg/capr"
	"github.com/rancher/rancher/pkg/capr/mock/mockcapicontrollers"
	"github.com/rancher/rancher/pkg/capr/mock/mockcorecontrollers"
	"github.com/rancher/rancher/pkg/capr/mock/mockmgmtcontrollers"
	"github.com/rancher/rancher/pkg/capr/mock/mockranchercontrollers"
	"github.com/rancher/rancher/pkg/capr/mock/mockrkecontrollers"
	"github.com/rancher/rancher/pkg/provisioningv2/image"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
)

type mockPlanner struct {
	planner                       *Planner
	rkeBootstrap                  *mockrkecontrollers.MockRKEBootstrapClient
	rkeBootstrapCache             *mockrkecontrollers.MockRKEBootstrapCache
	rkeControlPlanes              *mockrkecontrollers.MockRKEControlPlaneController
	etcdSnapshotCache             *mockrkecontrollers.MockETCDSnapshotCache
	secretClient                  *mockcorecontrollers.MockSecretClient
	secretCache                   *mockcorecontrollers.MockSecretCache
	configMapCache                *mockcorecontrollers.MockConfigMapCache
	machines                      *mockcapicontrollers.MockMachineClient
	machinesCache                 *mockcapicontrollers.MockMachineCache
	clusterRegistrationTokenCache *mockmgmtcontrollers.MockClusterRegistrationTokenCache
	capiClient                    *mockcapicontrollers.MockClusterClient
	capiClusters                  *mockcapicontrollers.MockClusterCache
	managementClusters            *mockmgmtcontrollers.MockClusterCache
	rancherClusterCache           *mockranchercontrollers.MockClusterCache
}

// newMockPlanner creates a new mockPlanner that can be used for simulating a functional Planner.
func newMockPlanner(t *testing.T, functions InfoFunctions) *mockPlanner {
	ctrl := gomock.NewController(t)
	mp := mockPlanner{
		rkeBootstrap:                  mockrkecontrollers.NewMockRKEBootstrapClient(ctrl),
		rkeBootstrapCache:             mockrkecontrollers.NewMockRKEBootstrapCache(ctrl),
		rkeControlPlanes:              mockrkecontrollers.NewMockRKEControlPlaneController(ctrl),
		etcdSnapshotCache:             mockrkecontrollers.NewMockETCDSnapshotCache(ctrl),
		secretClient:                  mockcorecontrollers.NewMockSecretClient(ctrl),
		secretCache:                   mockcorecontrollers.NewMockSecretCache(ctrl),
		configMapCache:                mockcorecontrollers.NewMockConfigMapCache(ctrl),
		machines:                      mockcapicontrollers.NewMockMachineClient(ctrl),
		machinesCache:                 mockcapicontrollers.NewMockMachineCache(ctrl),
		clusterRegistrationTokenCache: mockmgmtcontrollers.NewMockClusterRegistrationTokenCache(ctrl),
		capiClient:                    mockcapicontrollers.NewMockClusterClient(ctrl),
		capiClusters:                  mockcapicontrollers.NewMockClusterCache(ctrl),
		managementClusters:            mockmgmtcontrollers.NewMockClusterCache(ctrl),
		rancherClusterCache:           mockranchercontrollers.NewMockClusterCache(ctrl),
	}
	store := PlanStore{
		secrets:      mp.secretClient,
		secretsCache: mp.secretCache,
		machineCache: mp.machinesCache,
	}
	p := Planner{
		ctx:                           context.TODO(),
		store:                         &store,
		machines:                      mp.machines,
		machinesCache:                 mp.machinesCache,
		secretClient:                  mp.secretClient,
		secretCache:                   mp.secretCache,
		configMapCache:                mp.configMapCache,
		clusterRegistrationTokenCache: mp.clusterRegistrationTokenCache,
		capiClient:                    mp.capiClient,
		capiClusters:                  mp.capiClusters,
		managementClusters:            mp.managementClusters,
		rancherClusterCache:           mp.rancherClusterCache,
		rkeControlPlanes:              mp.rkeControlPlanes,
		rkeBootstrap:                  mp.rkeBootstrap,
		rkeBootstrapCache:             mp.rkeBootstrapCache,
		etcdSnapshotCache:             mp.etcdSnapshotCache,
		etcdS3Args: s3Args{
			secretCache: mp.secretCache,
		},
		retrievalFunctions: functions,
	}
	mp.planner = &p
	return &mp
}

func TestPlanner_addInstruction(t *testing.T) {
	type args struct {
		version         string
		expectedVersion string
		os              string
		command         string
		scriptName      string
		envs            []string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Checking Linux Instructions",
			args: args{
				version:         "v1.21.5+rke2r2",
				expectedVersion: "v1.21.5-rke2r2",
				os:              "linux",
				command:         "sh",
				scriptName:      "run.sh",
				envs:            []string{"INSTALL_RKE2_EXEC"},
			},
		},
		{
			name: "Checking Windows Instructions",
			args: args{
				version:         "v1.21.5+rke2r2",
				expectedVersion: "v1.21.5-rke2r2",
				os:              "windows",
				command:         "powershell.exe",
				scriptName:      "run.ps1",
				envs:            []string{"$env:RESTART_STAMP", "$env:INSTALL_RKE2_EXEC"},
			},
		},
		{
			name: "Checking K3s Instructions",
			args: args{
				version:         "v1.21.5+k3s2",
				expectedVersion: "v1.21.5-k3s2",
				os:              "linux",
				command:         "sh",
				scriptName:      "run.sh",
				envs:            []string{"INSTALL_K3S_EXEC"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			a := assert.New(t)
			var planner Planner
			controlPlane := createTestControlPlane(tt.args.version)
			entry := createTestPlanEntry(tt.args.os)
			planner.retrievalFunctions.SystemAgentImage = func() string { return "system-agent" }
			planner.retrievalFunctions.ImageResolver = image.ResolveWithControlPlane
			// act
			p, err := planner.addInstallInstructionWithRestartStamp(plan.NodePlan{}, controlPlane, entry)

			// assert
			a.Nil(err)
			a.NotNil(p)
			a.Equal(entry.Metadata.Labels[capr.CattleOSLabel], tt.args.os)
			a.NotZero(len(p.Instructions))
			instruction := p.Instructions[0]
			a.Contains(instruction.Command, tt.args.command)
			a.Contains(instruction.Image, tt.args.expectedVersion)
			a.Contains(instruction.Args, tt.args.scriptName)
			for _, e := range tt.args.envs {
				a.True(findEnv(instruction.Env, e), "couldn't find %s in environment", e)
			}
		})
	}
}

func createTestControlPlane(version string) *rkev1.RKEControlPlane {
	return &rkev1.RKEControlPlane{
		Spec: rkev1.RKEControlPlaneSpec{
			KubernetesVersion: version,
		},
	}
}

func createTestPlanEntry(os string) *planEntry {
	return &planEntry{
		Machine: &capi.Machine{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					capr.ControlPlaneRoleLabel: "false",
					capr.EtcdRoleLabel:         "false",
					capr.WorkerRoleLabel:       "true",
				},
			},
			Spec: capi.MachineSpec{},
			Status: capi.MachineStatus{
				NodeInfo: &v1.NodeSystemInfo{
					OperatingSystem: os,
				},
			},
		},
		Metadata: &plan.Metadata{
			Labels: map[string]string{
				capr.CattleOSLabel:         os,
				capr.ControlPlaneRoleLabel: "false",
				capr.EtcdRoleLabel:         "false",
				capr.WorkerRoleLabel:       "true",
			},
		},
	}
}

func createTestPlanEntryWithoutRoles(os string) *planEntry {
	entry := createTestPlanEntry(os)
	entry.Metadata.Labels = map[string]string{
		capr.CattleOSLabel: os,
	}
	return entry
}

func findEnv(s []string, v string) bool {
	for _, item := range s {
		if strings.Contains(item, v) {
			return true
		}
	}
	return false
}

func Test_IsWindows(t *testing.T) {
	a := assert.New(t)
	data := map[string]bool{
		"windows": true,
		"linux":   false,
		"":        false,
	}
	for k, v := range data {
		a.Equal(v, windows(&planEntry{
			Metadata: &plan.Metadata{
				Labels: map[string]string{
					capr.CattleOSLabel: k,
				},
			},
		}))
	}
}

func Test_notWindows(t *testing.T) {
	type args struct {
		entry    *planEntry
		expected bool
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Checking that linux isn't windows",
			args: args{
				entry:    createTestPlanEntry("linux"),
				expected: true,
			},
		},
		{
			name: "Checking that windows is windows",
			args: args{
				entry:    createTestPlanEntry("windows"),
				expected: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			a := assert.New(t)

			// act
			result := roleNot(windows)(tt.args.entry)

			// assert
			a.Equal(result, tt.args.expected)
		})
	}
}

func Test_anyRoleWithoutWindows(t *testing.T) {
	type args struct {
		entry    *planEntry
		expected bool
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Should return linux node with roles",
			args: args{
				entry:    createTestPlanEntry("linux"),
				expected: true,
			},
		},
		{
			name: "Shouldn't return windows node.",
			args: args{
				entry:    createTestPlanEntry("windows"),
				expected: false,
			},
		},
		{
			name: "Shouldn't return node without any roles.",
			args: args{
				entry:    createTestPlanEntryWithoutRoles("linux"),
				expected: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			a := assert.New(t)

			// act
			result := anyRoleWithoutWindows(tt.args.entry)

			// assert
			a.Equal(result, tt.args.expected)
		})
	}
}

func TestPlanner_getLowestMachineKubeletVersion(t *testing.T) {
	type args struct {
		versions       []string
		expectedLowest string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Check lowest RKE2 version within minor release",
			args: args{
				versions: []string{
					"v1.25.5+rke2r1",
					"v1.25.6+rke2r1",
					"v1.25.7+rke2r1",
				},
				expectedLowest: "v1.25.5+rke2r1",
			},
		},
		{
			name: "Check lowest K3s version within minor release",
			args: args{
				versions: []string{
					"v1.25.5+k3s1",
					"v1.25.6+k3s1",
					"v1.25.7+k3s1",
				},
				expectedLowest: "v1.25.5+k3s1",
			},
		},
		{
			name: "Check lowest RKE2 version across any change in release",
			args: args{
				versions: []string{
					"v1.25.4+rke2r1",
					"v2.21.6+rke2r1",
					"v1.26.7+rke2r1",
				},
				expectedLowest: "v1.25.4+rke2r1",
			},
		},
		{
			name: "Check lowest K3s version across any change in release",
			args: args{
				versions: []string{
					"v1.25.4+k3s1",
					"v2.21.6+k3s1",
					"v1.26.7+k3s1",
				},
				expectedLowest: "v1.25.4+k3s1",
			},
		},
		{
			name: "Check lowest version across mixed K3s/RKE2 cluster",
			args: args{
				versions: []string{
					"v1.25.4+k3s1",
					"v2.21.6+k3s1",
					"v1.26.7+k3s1",
					"v1.21.5+rke2r1",
				},
				expectedLowest: "v1.21.5+rke2r1",
			},
		},
		{
			name: "Check lowest K3s version with RCs",
			args: args{
				versions: []string{
					"v1.21.4+k3s1",
					"v1.21.3-rc1+k3s1",
					"v1.23.7+k3s1",
				},
				expectedLowest: "v1.21.3-rc1+k3s1",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			var plan = &plan.Plan{
				Machines: map[string]*capi.Machine{},
			}
			rand.Seed(time.Now().UnixNano())
			versions := test.args.versions
			// Shuffle the versions to really test the function.
			rand.Shuffle(len(versions), func(i, j int) { versions[i], versions[j] = versions[j], versions[i] })
			for i, v := range versions {
				plan.Machines[fmt.Sprintf("machine%d", i)] = &capi.Machine{
					Status: capi.MachineStatus{
						NodeInfo: &v1.NodeSystemInfo{
							KubeletVersion: v,
						},
					},
				}
			}
			lowestV := getLowestMachineKubeletVersion(plan)
			if len(test.args.versions) > 0 {
				a.NotNil(lowestV)
				expectedLowest, err := semver.NewVersion(test.args.expectedLowest)
				if a.NoError(err) {
					a.Equal(lowestV.String(), expectedLowest.String())
				}
			} else {
				a.Nil(lowestV)
			}
		})
	}
}

func Test_getInstallerImage(t *testing.T) {
	tests := []struct {
		name         string
		expected     string
		controlPlane *rkev1.RKEControlPlane
	}{
		{
			name:     "default",
			expected: "rancher/system-agent-installer-rke2:v1.25.7-rke2r1",
			controlPlane: &rkev1.RKEControlPlane{
				Spec: rkev1.RKEControlPlaneSpec{
					KubernetesVersion: "v1.25.7+rke2r1",
				},
			},
		},
		{
			name:     "cluster private registry - machine global",
			expected: "test.rancher.io/rancher/system-agent-installer-rke2:v1.25.7-rke2r1",
			controlPlane: &rkev1.RKEControlPlane{
				Spec: rkev1.RKEControlPlaneSpec{
					RKEClusterSpecCommon: rkev1.RKEClusterSpecCommon{
						MachineGlobalConfig: rkev1.GenericMap{
							Data: map[string]any{
								"system-default-registry": "test.rancher.io",
							},
						},
					},
					KubernetesVersion: "v1.25.7+rke2r1",
				},
			},
		},
		{
			name:     "cluster private registry - machine selector",
			expected: "test.rancher.io/rancher/system-agent-installer-rke2:v1.25.7-rke2r1",
			controlPlane: &rkev1.RKEControlPlane{
				Spec: rkev1.RKEControlPlaneSpec{
					RKEClusterSpecCommon: rkev1.RKEClusterSpecCommon{
						MachineSelectorConfig: []rkev1.RKESystemConfig{
							{
								Config: rkev1.GenericMap{
									Data: map[string]any{
										"system-default-registry": "test.rancher.io",
									},
								},
							},
						},
					},
					KubernetesVersion: "v1.25.7+rke2r1",
				},
			},
		},
		{
			name:     "cluster private registry - prefer machine global",
			expected: "test.rancher.io/rancher/system-agent-installer-rke2:v1.25.7-rke2r1",
			controlPlane: &rkev1.RKEControlPlane{
				Spec: rkev1.RKEControlPlaneSpec{
					RKEClusterSpecCommon: rkev1.RKEClusterSpecCommon{
						MachineGlobalConfig: rkev1.GenericMap{
							Data: map[string]any{
								"system-default-registry": "test.rancher.io",
							},
						},
						MachineSelectorConfig: []rkev1.RKESystemConfig{
							{
								Config: rkev1.GenericMap{
									Data: map[string]any{
										"system-default-registry": "test2.rancher.io",
									},
								},
							},
						},
					},
					KubernetesVersion: "v1.25.7+rke2r1",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var planner Planner
			planner.retrievalFunctions.ImageResolver = image.ResolveWithControlPlane
			planner.retrievalFunctions.SystemAgentImage = func() string { return "rancher/system-agent-installer-" }

			assert.Equal(t, tt.expected, planner.getInstallerImage(tt.controlPlane))
		})
	}
}

func Test_renderArgAndMount(t *testing.T) {
	tests := []struct {
		name            string
		inputArg        interface{}
		inputMount      interface{}
		inputSecurePort string
		inputCertDir    string
		runtime         string
		expectedArgs    []string
		expectedMount   []string
	}{
		{
			name:            "test default K3s KCM rendering",
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{CertDirArgument + "=" + fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeK3S), SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test custom K3s KCM cert-dir",
			inputArg:        "cert-dir=/tmp",
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{"cert-dir=/tmp", SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test custom K3s KCM tls-cert-file",
			inputArg:        "tls-cert-file=/mycustomfile.crt",
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{"tls-cert-file=/mycustomfile.crt", SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test custom K3s KCM cert-dir with surrounding bogus data in input args",
			inputArg:        []string{"bogus", "cert-dir=/tmp", "data:"},
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{"bogus", "cert-dir=/tmp", "data:", SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test custom K3s KCM tls-cert-file with surrounding bogus data in input args",
			inputArg:        []string{"bogus=", "tls-cert-file=/mycustomfile.crt", "data"},
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{"bogus=", "tls-cert-file=/mycustomfile.crt", "data", SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test default RKE2 KCM rendering",
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{CertDirArgument + "=" + fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2), SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2) + ":" + fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2)},
		},
		{
			name:            "test custom RKE2 KCM cert-dir",
			inputArg:        "cert-dir=/tmp",
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"cert-dir=/tmp", SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{"/tmp:/tmp"},
		},
		{
			name:            "test custom RKE2 KCM tls-cert-file",
			inputArg:        "tls-cert-file=/somedir/mycustomfile.crt",
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"tls-cert-file=/somedir/mycustomfile.crt", SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{"/somedir:/somedir"},
		},
		{
			name:            "test custom RKE2 KCM cert-dir with surrounding bogus data in input args",
			inputArg:        []string{"bogus", "cert-dir=/tmp", "data:"},
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"bogus", "cert-dir=/tmp", "data:", SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{"/tmp:/tmp"},
		},
		{
			name:            "test custom RKE2 KCM tls-cert-file with surrounding bogus data in input args",
			inputArg:        []string{"bogus=", "tls-cert-file=/mycustomfile.crt", "data"},
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"bogus=", "tls-cert-file=/mycustomfile.crt", "data", SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{"/:/"}, // this is notably going to break things but it's still a good demonstration of expected value. If we ever add a validation for this in the future we need to change this test.
		},
		{
			name:            "test custom RKE2 KCM empty tls-cert-file with surrounding bogus data in input args",
			inputArg:        []string{"tls-cert-file=", "data"},
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"tls-cert-file=", "data", CertDirArgument + "=" + fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2), SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2) + ":" + fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2)},
		},
		{
			name:            "test custom RKE2 KCM empty cert-dir with surrounding bogus data in input args",
			inputArg:        []string{"cert-dir=", "data"},
			inputSecurePort: DefaultKubeControllerManagerDefaultSecurePort,
			inputCertDir:    DefaultKubeControllerManagerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"cert-dir=", "data", CertDirArgument + "=" + fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2), SecurePortArgument + "=" + DefaultKubeControllerManagerDefaultSecurePort},
			expectedMount:   []string{fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2) + ":" + fmt.Sprintf(DefaultKubeControllerManagerCertDir, capr.RuntimeRKE2)},
		},
		{
			name:            "test default K3s kube-scheduler rendering",
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{CertDirArgument + "=" + fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeK3S), SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test custom K3s kube-scheduler cert-dir",
			inputArg:        "cert-dir=/tmp",
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{"cert-dir=/tmp", SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test custom K3s kube-scheduler tls-cert-file",
			inputArg:        "tls-cert-file=/mycustomfile.crt",
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{"tls-cert-file=/mycustomfile.crt", SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test custom K3s kube-scheduler cert-dir with surrounding bogus data in input args",
			inputArg:        []string{"bogus", "cert-dir=/tmp", "data:"},
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{"bogus", "cert-dir=/tmp", "data:", SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test custom K3s kube-scheduler tls-cert-file with surrounding bogus data in input args",
			inputArg:        []string{"bogus=", "tls-cert-file=/mycustomfile.crt", "data"},
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeK3S,
			expectedArgs:    []string{"bogus=", "tls-cert-file=/mycustomfile.crt", "data", SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{},
		},
		{
			name:            "test default RKE2 kube-scheduler rendering",
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{CertDirArgument + "=" + fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2), SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2) + ":" + fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2)},
		},
		{
			name:            "test custom RKE2 kube-scheduler cert-dir",
			inputArg:        "cert-dir=/tmp",
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"cert-dir=/tmp", SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{"/tmp:/tmp"},
		},
		{
			name:            "test custom RKE2 kube-scheduler tls-cert-file",
			inputArg:        "tls-cert-file=/somedir/mycustomfile.crt",
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"tls-cert-file=/somedir/mycustomfile.crt", SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{"/somedir:/somedir"},
		},
		{
			name:            "test custom RKE2 kube-scheduler cert-dir with surrounding bogus data in input args",
			inputArg:        []string{"bogus", "cert-dir=/tmp", "data:"},
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"bogus", "cert-dir=/tmp", "data:", SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{"/tmp:/tmp"},
		},
		{
			name:            "test custom RKE2 kube-scheduler tls-cert-file with surrounding bogus data in input args",
			inputArg:        []string{"bogus=", "tls-cert-file=/mycustomfile.crt", "data"},
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"bogus=", "tls-cert-file=/mycustomfile.crt", "data", SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{"/:/"}, // this is notably going to break things but it's still a good demonstration of expected value. If we ever add a validation for this in the future we need to change this test.
		},
		{
			name:            "test custom RKE2 kube-scheduler empty tls-cert-file with surrounding bogus data in input args",
			inputArg:        []string{"tls-cert-file=", "data"},
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"tls-cert-file=", "data", CertDirArgument + "=" + fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2), SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2) + ":" + fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2)},
		},
		{
			name:            "test custom RKE2 kube-scheduler empty cert-dir with surrounding bogus data in input args",
			inputArg:        []string{"cert-dir=", "data"},
			inputSecurePort: DefaultKubeSchedulerDefaultSecurePort,
			inputCertDir:    DefaultKubeSchedulerCertDir,
			runtime:         capr.RuntimeRKE2,
			expectedArgs:    []string{"cert-dir=", "data", CertDirArgument + "=" + fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2), SecurePortArgument + "=" + DefaultKubeSchedulerDefaultSecurePort},
			expectedMount:   []string{fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2) + ":" + fmt.Sprintf(DefaultKubeSchedulerCertDir, capr.RuntimeRKE2)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, mounts := renderArgAndMount(tt.inputArg, tt.inputMount, tt.runtime, tt.inputSecurePort, tt.inputCertDir)
			assert.Equal(t, tt.expectedArgs, args, tt.name)
			assert.Equal(t, tt.expectedMount, mounts, tt.name)
		})
	}
}
