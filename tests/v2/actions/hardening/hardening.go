package hardening

import (
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Privileged = "privileged"
	Baseline   = "baseline"
	Restricted = "restricted"
)

var (
	RancherNamespaces = []string{
		"ingress-nginx",
		"kube-system",
		"cattle-system",
		"cattle-epinio-system",
		"cattle-fleet-system",
		"longhorn-system",
		"cattle-neuvector-system",
		"cattle-monitoring-system",
		"rancher-alerting-drivers",
		"cis-operator-system",
		"cattle-csp-adapter-system",
		"cattle-externalip-system",
		"cattle-gatekeeper-system",
		"istio-system",
		"cattle-istio-system",
		"cattle-logging-system",
		"cattle-windows-gmsa-system",
		"cattle-sriov-system",
		"cattle-ui-plugin-system",
		"tigera-operator",
	}
)

// CreateRancherBaselinePSACT creates custom PSACT called rancher-baseline which sets each PSS to baseline.
func CreateRancherPSACT(client *rancher.Client, psact, enforcement string, additionalNamespaces []string) error {
	_, err := client.Steve.SteveType(clusters.PodSecurityAdmissionSteveResoureType).ByID(psact)
	if err == nil {
		return nil
	}

	template := &v3.PodSecurityAdmissionConfigurationTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: psact,
		},
		Description: "This is a custom Pod Security Admission Configuration Template. " +
			"This policy contains namespace level exemptions for Rancher and custom-defined components.",
		Configuration: v3.PodSecurityAdmissionConfigurationTemplateSpec{
			Defaults: v3.PodSecurityAdmissionConfigurationTemplateDefaults{
				Enforce: enforcement,
				Audit:   enforcement,
				Warn:    enforcement,
			},
			Exemptions: v3.PodSecurityAdmissionConfigurationTemplateExemptions{
				Usernames:      []string{},
				RuntimeClasses: []string{},
				Namespaces:     append(RancherNamespaces, additionalNamespaces...),
			},
		},
	}

	_, err = client.Steve.SteveType(clusters.PodSecurityAdmissionSteveResoureType).Create(template)

	return err
}
