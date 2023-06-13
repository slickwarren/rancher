package hostedcluster

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	aksv1 "github.com/rancher/aks-operator/pkg/apis/aks.cattle.io/v1"
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/controllers/dashboard/chart"
	"github.com/rancher/rancher/pkg/controllers/dashboard/chart/fake"
	"github.com/rancher/rancher/pkg/settings"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var priorityClassName = "rancher-critical"

func Test_handler_onClusterChange(t *testing.T) {

	tests := []struct {
		name       string
		cluster    *v3.Cluster
		newManager func(ctrl *gomock.Controller) chart.Manager
		wantErr    bool
	}{
		{
			name: "normal installation",
			cluster: &v3.Cluster{
				Spec: v3.ClusterSpec{
					AKSConfig: &aksv1.AKSClusterConfigSpec{},
				},
			},
			newManager: func(ctrl *gomock.Controller) chart.Manager {
				settings.ConfigMapName.Set("pass")
				manager := fake.NewMockManager(ctrl)
				expectedValues := map[string]interface{}{
					"global": map[string]interface{}{
						"cattle": map[string]interface{}{
							"systemDefaultRegistry": settings.SystemDefaultRegistry.Get(),
						},
					},
					"httpProxy":            os.Getenv("HTTP_PROXY"),
					"httpsProxy":           os.Getenv("HTTPS_PROXY"),
					"noProxy":              os.Getenv("NO_PROXY"),
					"additionalTrustedCAs": false,
					"priorityClassName":    priorityClassName,
				}
				var b bool
				manager.EXPECT().Ensure(
					AksCrdChart.ReleaseNamespace,
					AksCrdChart.ChartName,
					settings.FleetMinVersion.Get(),
					"",
					nil,
					gomock.AssignableToTypeOf(b),
					"",
				).Return(nil)
				manager.EXPECT().Ensure(
					AksChart.ReleaseNamespace,
					AksChart.ChartName,
					settings.FleetMinVersion.Get(),
					"",
					expectedValues,
					gomock.AssignableToTypeOf(b),
					"",
				).Return(nil)

				return manager
			},
		},
		{
			name: "no priority class installation",
			cluster: &v3.Cluster{
				Spec: v3.ClusterSpec{
					AKSConfig: &aksv1.AKSClusterConfigSpec{},
				},
			},
			newManager: func(ctrl *gomock.Controller) chart.Manager {
				settings.ConfigMapName.Set("error")
				manager := fake.NewMockManager(ctrl)
				expectedValues := map[string]interface{}{
					"global": map[string]interface{}{
						"cattle": map[string]interface{}{
							"systemDefaultRegistry": settings.SystemDefaultRegistry.Get(),
						},
					},
					"httpProxy":            os.Getenv("HTTP_PROXY"),
					"httpsProxy":           os.Getenv("HTTPS_PROXY"),
					"noProxy":              os.Getenv("NO_PROXY"),
					"additionalTrustedCAs": false,
				}
				var b bool
				manager.EXPECT().Ensure(
					AksCrdChart.ReleaseNamespace,
					AksCrdChart.ChartName,
					settings.FleetMinVersion.Get(),
					"",
					nil,
					gomock.AssignableToTypeOf(b),
					"",
				).Return(nil)
				manager.EXPECT().Ensure(
					AksChart.ReleaseNamespace,
					AksChart.ChartName,
					settings.FleetMinVersion.Get(),
					"",
					expectedValues,
					gomock.AssignableToTypeOf(b),
					"",
				).Return(nil)

				return manager
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			h := newHandler(ctrl)
			h.manager = tt.newManager(ctrl)
			got, err := h.onClusterChange("", tt.cluster)
			if tt.wantErr {
				assert.Error(t, err, "handler.onRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.NoError(t, err, "unexpected error")
			if !reflect.DeepEqual(got, tt.cluster) {
				t.Errorf("handler.onClusterChange() = %v, want %v", got, tt.cluster)
			}
		})
	}
}

func newHandler(ctrl *gomock.Controller) *handler {
	appCache := NewMockAppCache(ctrl)
	appCache.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil)
	apps := NewMockAppController(ctrl)
	apps.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	projectCache := NewMockProjectCache(ctrl)
	projectCache.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*v3.Project{{ObjectMeta: metav1.ObjectMeta{Name: "test"}}}, nil)
	secretsCache := NewMockSecretCache(ctrl)
	secretsCache.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil)
	configCache := NewMockConfigMapCache(ctrl)
	configCache.EXPECT().Get(gomock.Any(), "pass").Return(&v1.ConfigMap{Data: map[string]string{"priorityClassName": priorityClassName}}, nil).AnyTimes()
	configCache.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("not found")).AnyTimes()
	return &handler{
		appCache:     appCache,
		apps:         apps,
		projectCache: projectCache,
		secretsCache: secretsCache,
		chartsConfig: chart.RancherConfigGetter{ConfigCache: configCache},
	}
}
