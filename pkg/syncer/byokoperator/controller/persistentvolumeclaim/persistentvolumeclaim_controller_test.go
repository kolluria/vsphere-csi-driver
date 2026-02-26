/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package persistentvolumeclaim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vmoperatortypes "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	csicommon "sigs.k8s.io/vsphere-csi-driver/v3/pkg/csi/service/common"
	"sigs.k8s.io/vsphere-csi-driver/v3/pkg/csi/service/logger"
)

// createTestScheme creates a scheme with all required types for testing
func createTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vmoperatortypes.AddToScheme(scheme)
	return scheme
}

// TestFindVolume tests the findVolume function
func TestFindVolume(t *testing.T) {
	ctx := context.Background()
	scheme := createTestScheme()
	volumeHandle := "test-volume-handle"
	pvName := "pv-test"

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: pvName,
		},
	}

	t.Run("Success", func(t *testing.T) {
		pv := &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver:       csicommon.VSphereCSIDriverName,
						VolumeHandle: volumeHandle,
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, pv).Build()
		r := &reconciler{
			Client: fakeClient,
			logger: logger.GetLoggerWithNoContext().Named("test"),
		}

		volID, err := r.findVolume(ctx, pvc)
		assert.NoError(t, err)
		assert.Equal(t, volumeHandle, volID)
	})

	t.Run("PVNotFound", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
		r := &reconciler{
			Client: fakeClient,
			logger: logger.GetLoggerWithNoContext().Named("test"),
		}

		volID, err := r.findVolume(ctx, pvc)
		assert.Error(t, err)
		assert.Empty(t, volID)
	})

	t.Run("NotVSphereCSIVolume", func(t *testing.T) {
		pv := &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver: "other-driver",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, pv).Build()
		r := &reconciler{
			Client: fakeClient,
			logger: logger.GetLoggerWithNoContext().Named("test"),
		}

		volID, err := r.findVolume(ctx, pvc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not a vSphere CSI volume")
		assert.Empty(t, volID)
	})

	t.Run("IsFileVolume", func(t *testing.T) {
		fileVolumeHandle := "file:test-volume-handle"
		pv := &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver:       csicommon.VSphereCSIDriverName,
						VolumeHandle: fileVolumeHandle,
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, pv).Build()
		r := &reconciler{
			Client: fakeClient,
			logger: logger.GetLoggerWithNoContext().Named("test"),
		}

		volID, err := r.findVolume(ctx, pvc)
		assert.NoError(t, err)
		assert.Empty(t, volID)
	})
}
