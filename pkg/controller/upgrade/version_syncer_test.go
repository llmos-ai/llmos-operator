package upgrade

import (
	"errors"
	"testing"
	"time"

	gversion "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
)

func Test_CheckUpgradableVersions(t *testing.T) {
	type input struct {
		versions       []mgmtv1.Version
		currentVersion string
	}
	type output struct {
		canUpgrade bool
		err        error
	}
	var testCases = []struct {
		name     string
		given    input
		expected []output
	}{
		{
			name: "bad versions",
			given: input{
				currentVersion: "v0.1.0",
				versions: []mgmtv1.Version{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "bad-version-123",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.1.0",
							Tags:                 []string{"dev"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.2.0",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "randomstring",
							Tags:                 []string{"dev"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.2.0-with-k8s",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.1.0",
							KubernetesVersion:    "abcv1.31.0+k3s1",
							Tags:                 []string{"dev"},
						},
					},
				},
			},
			expected: []output{
				{
					canUpgrade: false,
					err:        errors.New("Malformed version: bad-version-123"),
				},
				{
					canUpgrade: false,
					err:        errors.New("Malformed version: randomstring"),
				},
				{
					canUpgrade: false,
					err:        errors.New("Malformed version: abcv1.31.0+k3s1"),
				},
			},
		},
		{
			name: "common cases",
			given: input{
				currentVersion: "v0.1.0",
				versions: []mgmtv1.Version{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.0.0-dev",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.1.0",
							KubernetesVersion:    "v1.30.4+k3s1",
							Tags:                 []string{"dev"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.0.1",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.1.0",
							KubernetesVersion:    "v1.31.0+k3s1",
							Tags:                 []string{""},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.1.0-rc1",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.1.0",
							Tags:                 []string{"v0.1-rc"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.3.0",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.2.0",
							Tags:                 []string{"v0.3-latest"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.3.0-rc1",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.2.0", // dev tag should ignore this
							Tags:                 []string{"dev"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.3.0-rc1",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.2.0", // minUpgradableVersion is not met
						},
					},
				},
			},
			expected: []output{
				{canUpgrade: true},
				{canUpgrade: false},
				{canUpgrade: false},
				{canUpgrade: false},
				{canUpgrade: true},
				{canUpgrade: false},
			},
		},
		{
			name: "old versions",
			given: input{
				currentVersion: "v0.2.0",
				versions: []mgmtv1.Version{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.2.0-dev", // met by dev
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.1.0",
							KubernetesVersion:    "v1.30.4+k3s1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.2.0-rc1", // not met
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.1.0",
							KubernetesVersion:    "v1.30.4+k3s1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.2.0-rc2", // not met
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.2.0",
							KubernetesVersion:    "v1.30.4+k3s1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.1.8", // not met
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.2.0",
							KubernetesVersion:    "v1.30.4+k3s1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.1.8", // not met
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.2.1",
							KubernetesVersion:    "v1.30.4+k3s1",
							Tags:                 []string{"dev"}, // met by dev
						},
					},
				},
			},
			expected: []output{
				{canUpgrade: true},
				{canUpgrade: false},
				{canUpgrade: false},
				{canUpgrade: false},
				{canUpgrade: true},
			},
		},
		{
			name: "new versions",
			given: input{
				currentVersion: "v0.2.0",
				versions: []mgmtv1.Version{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.3.0-rc1",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.1.0",
							KubernetesVersion:    "v1.30.4+k3s1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.4.0",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.3.0", // not met
							KubernetesVersion:    "v1.30.4+k3s1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.4.0",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.2.1", // not met
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "v0.4.0",
						},
						Spec: mgmtv1.VersionSpec{
							ReleaseDate:          time.Now().String(),
							MinUpgradableVersion: "v0.2.0-rc1",
						},
					},
				},
			},
			expected: []output{
				{canUpgrade: true},
				{canUpgrade: false},
				{canUpgrade: false},
				{canUpgrade: true},
			},
		},
	}

	for _, tc := range testCases {
		var actual output
		currentVersion, err := gversion.NewSemver(tc.given.currentVersion)
		assert.Nil(t, err)

		for i := range tc.given.versions {
			actual.canUpgrade, actual.err = canUpgrade(currentVersion, &tc.given.versions[i])
			assert.Equal(t, tc.expected[i], actual, "case %q", tc.name)
		}
	}
}

func Test_ConvertToGi(t *testing.T) {
	testCases := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "test Mi",
			value: "32920204",
			want:  "32Mi",
		},
		{
			name:  "test Gi",
			value: "32920204Ki",
			want:  "32Gi",
		},
		{
			name:  "test Gi2",
			value: "32920204Mi",
			want:  "32149Gi",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := resource.ParseQuantity(tc.value)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, convertToGi(&q))
		})
	}
}
