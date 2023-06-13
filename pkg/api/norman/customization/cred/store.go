package cred

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/store/transform"
	"github.com/rancher/norman/types"
	"github.com/rancher/norman/types/convert"
	"github.com/rancher/rancher/pkg/api/norman/customization/namespacedresource"
	"github.com/rancher/rancher/pkg/controllers/provisioningv2/cluster"
	provv1 "github.com/rancher/rancher/pkg/generated/controllers/provisioning.cattle.io/v1"
	v1 "github.com/rancher/rancher/pkg/generated/norman/core/v1"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/namespace"
	"k8s.io/apimachinery/pkg/labels"
)

func Wrap(store types.Store, ns v1.NamespaceInterface, nodeTemplateLister v3.NodeTemplateLister, provClusterCache provv1.ClusterCache) types.Store {
	transformStore := &transform.Store{
		Store: store,
		Transformer: func(apiContext *types.APIContext, schema *types.Schema, data map[string]interface{}, opt *types.QueryOptions) (map[string]interface{}, error) {
			if configExists(data) {
				data["type"] = "cloudCredential"
				if err := decodeNonPasswordFields(data); err != nil {
					return nil, err
				}
				return data, nil
			}
			return nil, nil
		},
	}

	newStore := &Store{
		Store:              transformStore,
		NodeTemplateLister: nodeTemplateLister,
		ProvClusterCache:   provClusterCache,
	}

	return namespacedresource.Wrap(newStore, ns, namespace.GlobalNamespace)
}

func configExists(data map[string]interface{}) bool {
	for key, val := range data {
		if strings.HasSuffix(key, "Config") {
			if convert.ToString(val) != "" {
				return true
			}
		}
	}
	return false
}

func decodeNonPasswordFields(data map[string]interface{}) error {
	for key, val := range data {
		if strings.HasSuffix(key, "Config") {
			ans := convert.ToMapInterface(val)
			for field, value := range ans {
				decoded, err := base64.StdEncoding.DecodeString(convert.ToString(value))
				if err != nil {
					return err
				}
				ans[field] = string(decoded)
			}
		}
	}
	return nil
}

func Validator(request *types.APIContext, schema *types.Schema, data map[string]interface{}) error {
	if !configExists(data) {
		return httperror.NewAPIError(httperror.MissingRequired, "a Config field must be set")
	}

	return nil
}

type Store struct {
	types.Store
	NodeTemplateLister v3.NodeTemplateLister
	ProvClusterCache   provv1.ClusterCache
}

func (s *Store) Delete(apiContext *types.APIContext, schema *types.Schema, id string) (map[string]interface{}, error) {
	// make sure the credential isn't being used by an active RKE2/K3s cluster
	if provClusters, err := s.ProvClusterCache.GetByIndex(cluster.ByCloudCred, id); err != nil {
		return nil, httperror.NewAPIError(httperror.ServerError, fmt.Sprintf("An error was encountered while attempting to delete the cloud credential: %s", err))
	} else if len(provClusters) > 0 {
		return nil, httperror.NewAPIError(httperror.InvalidAction, fmt.Sprintf("Cloud credential is currently referenced by provisioning cluster %s", provClusters[0].Name))
	}

	// make sure the cloud credential isn't being used by an RKE1 node template
	// which may be used by an active cluster
	nodeTemplates, err := s.NodeTemplateLister.List("", labels.NewSelector())
	if err != nil {
		return nil, err
	}
	if len(nodeTemplates) > 0 {
		for _, template := range nodeTemplates {
			if template.Spec.CloudCredentialName != id {
				continue
			}
			return nil, httperror.NewAPIError(httperror.MethodNotAllowed, fmt.Sprintf("Cloud credential is currently referenced by node template %s", template.Name))
		}
	}
	return s.Store.Delete(apiContext, schema, id)
}
