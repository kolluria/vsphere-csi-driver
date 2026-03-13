package wcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/vsphere-csi-driver/v3/pkg/csi/service/common"
)

func TestGetSnapshotLimitForNamespace(t *testing.T) {
	// Save original functions and restore after tests
	originalGetConfig := getK8sConfig
	originalNewK8sClientFromConfig := newK8sClientFromConfig
	defer func() {
		getK8sConfig = originalGetConfig
		newK8sClientFromConfig = originalNewK8sClientFromConfig
	}()

	// Mock getK8sConfig to return a fake config
	getK8sConfig = func() (*restclient.Config, error) {
		return &restclient.Config{}, nil
	}

	t.Run("WhenConfigMapExists_ValidValue", func(t *testing.T) {
		// Setup
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ConfigMapCSILimits,
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				common.ConfigMapKeyMaxSnapshotsPerVolume: "5",
			},
		}
		fakeClient := fake.NewSimpleClientset(cm)
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return fakeClient, nil
		}

		// Execute
		limit, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify
		assert.Nil(t, err)
		assert.Equal(t, 5, limit)
	})

	t.Run("WhenConfigMapExists_ValueEqualsMax", func(t *testing.T) {
		// Setup
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ConfigMapCSILimits,
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				common.ConfigMapKeyMaxSnapshotsPerVolume: "32",
			},
		}
		fakeClient := fake.NewSimpleClientset(cm)
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return fakeClient, nil
		}

		// Execute
		limit, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify
		assert.Nil(t, err)
		assert.Equal(t, 32, limit)
	})

	t.Run("WhenConfigMapExists_ValueExceedsMax", func(t *testing.T) {
		// Setup
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ConfigMapCSILimits,
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				common.ConfigMapKeyMaxSnapshotsPerVolume: "50",
			},
		}
		fakeClient := fake.NewSimpleClientset(cm)
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return fakeClient, nil
		}

		// Execute
		limit, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify
		assert.Nil(t, err)
		assert.Equal(t, 32, limit) // Should be capped to absolute max
	})

	t.Run("WhenConfigMapExists_ValueIsZero", func(t *testing.T) {
		// Setup
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ConfigMapCSILimits,
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				common.ConfigMapKeyMaxSnapshotsPerVolume: "0",
			},
		}
		fakeClient := fake.NewSimpleClientset(cm)
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return fakeClient, nil
		}

		// Execute
		limit, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify
		assert.Nil(t, err)
		assert.Equal(t, 0, limit) // 0 means block all snapshots
	})

	t.Run("WhenConfigMapExists_ValueIsNegative", func(t *testing.T) {
		// Setup
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ConfigMapCSILimits,
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				common.ConfigMapKeyMaxSnapshotsPerVolume: "-5",
			},
		}
		fakeClient := fake.NewSimpleClientset(cm)
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return fakeClient, nil
		}

		// Execute
		_, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid value")
		assert.Contains(t, err.Error(), "must be a non-negative integer")
	})

	t.Run("WhenConfigMapExists_InvalidFormat", func(t *testing.T) {
		// Setup
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ConfigMapCSILimits,
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				common.ConfigMapKeyMaxSnapshotsPerVolume: "abc",
			},
		}
		fakeClient := fake.NewSimpleClientset(cm)
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return fakeClient, nil
		}

		// Execute
		_, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid value")
		assert.Contains(t, err.Error(), "must be a non-negative integer")
	})

	t.Run("WhenConfigMapExists_MissingKey", func(t *testing.T) {
		// Setup
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ConfigMapCSILimits,
				Namespace: "test-namespace",
			},
			Data: map[string]string{}, // ConfigMap exists but key is missing
		}
		fakeClient := fake.NewSimpleClientset(cm)
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return fakeClient, nil
		}

		// Execute
		_, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "missing required key")
	})

	t.Run("WhenConfigMapNotFound", func(t *testing.T) {
		// Setup
		fakeClient := fake.NewSimpleClientset() // Empty clientset
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return fakeClient, nil
		}

		// Execute
		limit, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify
		assert.Nil(t, err)
		assert.Equal(t, common.DefaultMaxSnapshotsPerVolume, limit) // Should return default (4)
	})

	t.Run("WhenK8sClientCreationFails", func(t *testing.T) {
		// Setup
		newK8sClientFromConfig = func(c *restclient.Config) (kubernetes.Interface, error) {
			return nil, assert.AnError
		}

		// Execute
		limit, err := getSnapshotLimitForNamespace(context.Background(), "test-namespace")

		// Verify - should return default instead of error
		assert.Nil(t, err)
		assert.Equal(t, common.DefaultMaxSnapshotsPerVolume, limit)
	})
}
