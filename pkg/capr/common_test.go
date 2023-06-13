package capr

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	rkev1 "github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1"
	capicontrollers "github.com/rancher/rancher/pkg/generated/controllers/cluster.x-k8s.io/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
)

// Implements ClusterCache
type test struct {
	name        string
	expected    *capi.Cluster
	expectedErr error
	obj         runtime.Object
}

func (t *test) Get(_, _ string) (*capi.Cluster, error) {
	return t.expected, t.expectedErr
}

func (t *test) List(_ string, _ labels.Selector) ([]*capi.Cluster, error) {
	panic("not implemented")
}

func (t *test) AddIndexer(_ string, _ capicontrollers.ClusterIndexer) {
	panic("not implemented")
}

func (t *test) GetByIndex(_, _ string) ([]*capi.Cluster, error) {
	panic("not implemented")
}

func TestFindCAPIClusterFromLabel(t *testing.T) {
	tests := []test{
		{
			name:        "nil",
			expected:    nil,
			expectedErr: errNilObject,
			obj:         nil,
		},
		{
			name:        "missing label",
			expected:    nil,
			expectedErr: errors.New("cluster.x-k8s.io/cluster-name label not present on testObject: testNamespace/testName"),
			obj: &capi.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testName",
					Namespace: "testNamespace",
					Labels:    map[string]string{},
				},
				TypeMeta: metav1.TypeMeta{Kind: "testObject"},
			},
		},
		{
			name:     "missing cluster",
			expected: nil,
			obj: &capi.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"cluster.x-k8s.io/cluster-name": "testCluster",
					},
				},
			},
		},
		{
			name:        "success",
			expected:    &capi.Cluster{},
			expectedErr: nil,
			obj: &capi.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"cluster.x-k8s.io/cluster-name": "testCluster",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster, err := GetCAPIClusterFromLabel(tt.obj, &tt)
			if err == nil {
				assert.Nil(t, tt.expectedErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.Fail(t, "expected err to be nil, was actually %s", err)
			}
			assert.Equal(t, tt.expected, cluster)
		})
	}
}

func TestFindOwnerCAPICluster(t *testing.T) {
	tests := []test{
		{
			name:        "nil",
			expected:    nil,
			expectedErr: errNilObject,
			obj:         nil,
		},
		{
			name:        "no owner",
			expected:    nil,
			expectedErr: ErrNoMatchingControllerOwnerRef,
			obj: &rkev1.RKECluster{
				TypeMeta: metav1.TypeMeta{
					Kind: "RKECluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testcluster",
					Namespace: "testnamespace",
				},
			},
		},
		{
			name:        "no controller",
			expected:    nil,
			expectedErr: ErrNoMatchingControllerOwnerRef,
			obj: &rkev1.RKECluster{
				TypeMeta: metav1.TypeMeta{
					Kind: "RKECluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testcluster",
					Namespace: "testnamespace",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "nil",
						},
					},
				},
			},
		},
		{
			name:        "owner wrong kind",
			expected:    nil,
			expectedErr: ErrNoMatchingControllerOwnerRef,
			obj: &rkev1.RKECluster{
				TypeMeta: metav1.TypeMeta{
					Kind: "RKECluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testcluster",
					Namespace: "testnamespace",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "nil",
							APIVersion: "cluster.x-k8s.io/v1beta1",
							Controller: &[]bool{true}[0],
						},
					},
				},
			},
		},
		{
			name:        "owner wrong api version",
			expected:    nil,
			expectedErr: ErrNoMatchingControllerOwnerRef,
			obj: &rkev1.RKECluster{
				TypeMeta: metav1.TypeMeta{
					Kind: "RKECluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testcluster",
					Namespace: "testnamespace",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "Cluster",
							APIVersion: "nil",
							Controller: &[]bool{true}[0],
						},
					},
				},
			},
		},
		{
			name:        "success",
			expected:    nil,
			expectedErr: nil,
			obj: &rkev1.RKECluster{
				TypeMeta: metav1.TypeMeta{
					Kind: "RKECluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testcluster",
					Namespace: "testnamespace",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "Cluster",
							APIVersion: "cluster.x-k8s.io/v1beta1",
							Controller: &[]bool{true}[0],
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster, err := GetOwnerCAPICluster(tt.obj, &tt)
			if err == nil {
				assert.Nil(t, tt.expectedErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.Fail(t, "expected err to be nil, was actually %s", err)
			}
			assert.Equal(t, tt.expected, cluster)
		})
	}
}

func TestSafeConcatName(t *testing.T) {

	testcase := []struct {
		name           string
		input          []string
		expectedOutput string
		maxLength      int
	}{
		{
			name:           "max k8s name shortening",
			input:          []string{"very", "long", "name", "to", "test", "shortening", "behavior", "this", "should", "exceed", "max", "k8s", "name", "length"},
			expectedOutput: "very-long-name-to-test-shortening-behavior-this-should-ex-e8118",
			maxLength:      63,
		},
		{
			name:           "max helm release name shortening",
			maxLength:      53,
			input:          []string{"long", "cluster", "name", "testing", "managed", "system-upgrade", "controller", "fleet", "agent"},
			expectedOutput: "long-cluster-name-testing-managed-system-upgrad-0beef",
		},
		{
			name:           "max length smaller than hash size should concat and shorten but not hash",
			maxLength:      3,
			input:          []string{"this", "will", "not", "be", "hashed"},
			expectedOutput: "thi",
		},
		{
			name:           "concat but not shorten",
			maxLength:      90,
			input:          []string{"simple", "concat", "no", "hash", "needed"},
			expectedOutput: "simple-concat-no-hash-needed",
		},
		{
			name:           "no max length, no output",
			maxLength:      0,
			input:          []string{"input"},
			expectedOutput: "",
		},
		{
			name:           "input equal to hash length should return hash without leading '-'",
			maxLength:      6,
			input:          []string{"input", "s"},
			expectedOutput: "deab5",
		},
		{
			name:           "avoid special characters",
			maxLength:      8,
			input:          []string{"a", "&", "b", "=", "c"},
			expectedOutput: "a-359087",
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			out := SafeConcatName(tc.maxLength, tc.input...)
			if len(out) > tc.maxLength || out != tc.expectedOutput {
				t.Fail()
				t.Logf("expected output %s with length of %d, got %s with length of %d", tc.expectedOutput, len(tc.expectedOutput), out, len(out))
			}
			t.Log(out)
		})
	}
}

func TestCompressInterface(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "int",
			value: &[]int{1}[0],
		},
		{
			name:  "string",
			value: &[]string{"test"}[0],
		},
		{
			name: "struct",
			value: &struct {
				TestInt    int
				TestString string
			}{
				TestInt:    1,
				TestString: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompressInterface(tt.value)
			assert.Nil(t, err)
			assert.True(t, result != "")

			target := reflect.New(reflect.ValueOf(tt.value).Elem().Type()).Interface()

			err = DecompressInterface(result, target)
			assert.Nil(t, err)
			assert.Equal(t, tt.value, target)
		})
	}
}
