package reconcilehelper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCopyStatefulSetFields(t *testing.T) {
	// Test case 1: No update required (identical StatefulSets)
	t.Run("No Update Needed", func(t *testing.T) {
		now := time.Now()
		// Create a statefulset that is identical
		from := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"timestamp": now.UTC().Format("2024-01-02T15:04:05Z"),
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointerInt32(1),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"key": "value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container",
								Image: "nginx:latest",
							},
						},
					},
				},
			},
		}
		to := from.DeepCopy()

		requireUpdate, requireRedeploy := CopyStatefulSetFields(from, to)

		// Expect no updates needed and no redeployment
		assert.False(t, requireUpdate)
		assert.False(t, requireRedeploy)
	})

	// Test case 2: Update required (replicas differ)
	t.Run("Replicas Update", func(t *testing.T) {
		from := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointerInt32(3),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container",
								Image: "nginx:latest",
							},
						},
					},
				},
			},
		}
		to := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointerInt32(2),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container",
								Image: "nginx:latest",
							},
						},
					},
				},
			},
		}

		requireUpdate, requireRedeploy := CopyStatefulSetFields(from, to)

		// Expect an update due to different replica count
		assert.True(t, requireUpdate)
		assert.False(t, requireRedeploy)
	})

	// Test case 3: Redeploy required (container image changed)
	t.Run("Redeploy Needed (Container Image Changed)", func(t *testing.T) {
		from := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointerInt32(1),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container",
								Image: "nginx:1.19.0",
							},
						},
					},
				},
			},
		}

		to := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointerInt32(1),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container",
								Image: "nginx:latest", // Different image
							},
						},
					},
				},
			},
		}

		requireUpdate, requireRedeploy := CopyStatefulSetFields(from, to)

		// Expect redeployment due to image change
		assert.True(t, requireRedeploy)
		assert.True(t, requireUpdate) // Image change should trigger update
	})

	// Test case 4: Annotations need to be updated (timestamps)
	t.Run("Annotations Updated", func(t *testing.T) {
		from := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"timestamp": "2024-12-18T12:06:13Z",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointerInt32(1),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "container",
								Image:   "nginx:latest",
								Command: []string{"sleep", "infinity"},
							},
						},
					},
				},
			},
		}

		to := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"foo": "bar",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointerInt32(1),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "container",
								Image:   "nginx:latest",
								Command: []string{"sleep", "infinity2"},
							},
						},
					},
				},
			},
		}

		// Call the function
		requireUpdate, requireRedeploy := CopyStatefulSetFields(from, to)

		// Expect annotation to be copied and updated
		assert.True(t, requireUpdate)
		assert.True(t, requireRedeploy)
		assert.Equal(t, from.Annotations, to.Annotations)
	})
}

// Helper function to create pointer to int32
func pointerInt32(i int32) *int32 {
	return &i
}

// TestEqualIgnoreOrder covers a handful of scenarios where two string slices
// should be considered equal (ignoring order) or not.
func TestEqualIgnoreOrder(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{
			name: "both empty",
			a:    []string{},
			b:    []string{},
			want: true,
		},
		{
			name: "same elements, same order",
			a:    []string{"--task=embed", "--model=BAAI/bge-m3", "--tensor-parallel-size=1"},
			b:    []string{"--task=embed", "--model=BAAI/bge-m3", "--tensor-parallel-size=1"},
			want: true,
		},
		{
			name: "same elements, different order",
			a:    []string{"--task=embed", "--model=BAAI/bge-m3", "--tensor-parallel-size=1"},
			b:    []string{"--tensor-parallel-size=1", "--task=embed", "--model=BAAI/bge-m3"},
			want: true,
		},
		{
			name: "different elements",
			a:    []string{"--task=embed", "--model=BAAI/bge-m3"},
			b:    []string{"--task=embed", "--model=BAAI/other-model"},
			want: false,
		},
		{
			name: "different lengths",
			a:    []string{"--task=embed", "--model=BAAI/bge-m3"},
			b:    []string{"--task=embed"},
			want: false,
		},
		{
			name: "duplicates same count, different order",
			a:    []string{"--x", "--y", "--x"},
			b:    []string{"--x", "--x", "--y"},
			want: true,
		},
		{
			name: "duplicates different count",
			a:    []string{"--x", "--y", "--x"},
			b:    []string{"--x", "--y", "--y"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := equalIgnoreOrder(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("equalIgnoreOrder(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
