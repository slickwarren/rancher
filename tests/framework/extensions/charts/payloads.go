package charts

import (
	"time"

	"github.com/rancher/rancher/pkg/api/steve/catalog/types"
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newChartUninstallAction is a private constructor that creates a default payload for chart uninstall action with all disabled options.
func newChartUninstallAction() *types.ChartUninstallAction {
	return &types.ChartUninstallAction{
		DisableHooks: false,
		DryRun:       false,
		KeepHistory:  false,
		Timeout:      nil,
		Description:  "",
	}
}

// newChartInstallAction is a private constructor that creates a payload for chart install action with given namespace, projectId and chartInstalls.
func newChartInstallAction(namespace, projectId string, chartInstalls []types.ChartInstall) *types.ChartInstallAction {
	return &types.ChartInstallAction{
		DisableHooks:             false,
		Timeout:                  &metav1.Duration{Duration: 600 * time.Second},
		Wait:                     true,
		Namespace:                namespace,
		ProjectID:                projectId,
		DisableOpenAPIValidation: false,
		Charts:                   chartInstalls,
	}
}

// newChartUpgradeAction is a private constructor that creates a payload for chart upgrade action with given namespace and chartUpgrades.
func newChartUpgradeAction(namespace string, chartUpgrades []types.ChartUpgrade) *types.ChartUpgradeAction {
	return &types.ChartUpgradeAction{
		DisableHooks:             false,
		Timeout:                  &metav1.Duration{Duration: 600 * time.Second},
		Wait:                     true,
		Namespace:                namespace,
		DisableOpenAPIValidation: false,
		Force:                    false,
		CleanupOnFail:            false,
		Charts:                   chartUpgrades,
	}
}

// newChartInstallAction is a private constructor that creates a chart install with given chart values that can be used for chart install action.
func newChartInstall(name, version, clusterId, clusterName, url, defaultRegistry string, chartValues map[string]interface{}) *types.ChartInstall {
	chartInstall := types.ChartInstall{
		Annotations: map[string]string{
			"catalog.cattle.io/ui-source-repo":      "rancher-charts",
			"catalog.cattle.io/ui-source-repo-type": "cluster",
		},
		ChartName:   name,
		ReleaseName: name,
		Version:     version,
		Values: v3.MapStringInterface{
			"global": map[string]interface{}{
				"cattle": map[string]string{
					"clusterId":             clusterId,
					"clusterName":           clusterName,
					"rkePathPrefix":         "",
					"rkeWindowsPathPrefix":  "",
					"systemDefaultRegistry": defaultRegistry,
					"url":                   url,
				},
				"systemDefaultRegistry": defaultRegistry,
			},
		},
	}

	for k, v := range chartValues {
		chartInstall.Values[k] = v
	}

	return &chartInstall
}

// newChartUpgradeAction is a private constructor that creates a chart upgrade with given chart values that can be used for chart upgrade action.
func newChartUpgrade(name, version, clusterId, clusterName, url, defaultRegistry string, chartValues map[string]interface{}) *types.ChartUpgrade {
	chartUpgrade := types.ChartUpgrade{
		Annotations: map[string]string{
			"catalog.cattle.io/ui-source-repo":      "rancher-charts",
			"catalog.cattle.io/ui-source-repo-type": "cluster",
		},
		ChartName:   name,
		ReleaseName: name,
		Version:     version,
		Values: v3.MapStringInterface{
			"global": map[string]interface{}{
				"cattle": map[string]string{
					"clusterId":             clusterId,
					"clusterName":           clusterName,
					"rkePathPrefix":         "",
					"rkeWindowsPathPrefix":  "",
					"systemDefaultRegistry": defaultRegistry,
					"url":                   url,
				},
				"systemDefaultRegistry": defaultRegistry,
			},
		},
		ResetValues: false,
	}

	for k, v := range chartValues {
		chartUpgrade.Values[k] = v
	}

	return &chartUpgrade
}
