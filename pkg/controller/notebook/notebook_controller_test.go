package notebook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/fake"
	"github.com/llmos-ai/llmos-operator/pkg/utils/fakeclients"
)

func TestHandler_OnNotebookChanged(t *testing.T) {
	type input struct {
		key      string
		Notebook *mlv1.Notebook
	}
	type output struct {
		Notebook *mlv1.Notebook
		err      error
	}
	var testCases = []struct {
		name     string
		given    input
		expected output
	}{
		{
			name: "deleted resource",
			given: input{
				key: "default/nb-delete",
				Notebook: &mlv1.Notebook{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "default",
						Name:              "nb-delete",
						DeletionTimestamp: &metav1.Time{},
					},
				},
			},
			expected: output{
				Notebook: nil,
				err:      nil,
			},
		},
		{
			name: "create notebook",
			given: input{
				key: "default/nb-create",
				Notebook: &mlv1.Notebook{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "nb-create",
					},
					Spec: mlv1.NotebookSpec{
						Template: mlv1.NotebookTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "nb-create",
										Image: "busybox",
									},
								},
							},
						},
					},
				},
			},
			expected: output{
				Notebook: &mlv1.Notebook{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "nb-create",
					},
					Spec: mlv1.NotebookSpec{
						Template: mlv1.NotebookTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "nb-create",
										Image: "busybox",
									},
								},
							},
						},
					},
				},
				err: nil,
			},
		},
	}

	for _, tc := range testCases {
		fakeClient := fake.NewSimpleClientset()
		k8sClient := k8sfake.NewSimpleClientset()
		if tc.given.Notebook != nil {
			var err = fakeClient.Tracker().Add(tc.given.Notebook)
			assert.Nil(t, err, "mock resource should add into fake controller tracker")
		}

		h := &Handler{
			notebooks:        fakeclients.NotebookClient(fakeClient.MlV1().Notebooks),
			statefulSets:     fakeclients.StatefulSetClient(k8sClient.AppsV1().StatefulSets),
			statefulSetCache: fakeclients.StatefulSetCache(k8sClient.AppsV1().StatefulSets),
			services:         fakeclients.ServiceClient(k8sClient.CoreV1().Services),
			serviceCache:     fakeclients.ServiceCache(k8sClient.CoreV1().Services),
			podCache:         fakeclients.PodCache(k8sClient.CoreV1().Pods),
		}
		var actual output
		actual.Notebook, actual.err = h.OnChanged(tc.given.key, tc.given.Notebook)
		assert.Equal(t, tc.expected, actual, "case %q", tc.name)
	}
}
