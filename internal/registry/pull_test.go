package registry

import "testing"

func TestPullReference(t *testing.T) {
	tests := []struct {
		name         string
		registryHost string
		project      string
		image        string
		tag          string
		want         string
	}{
		{
			name:         "registry url host is normalized",
			registryHost: "https://registry.example.com",
			image:        "team/service",
			tag:          "v1.2.3",
			want:         "registry.example.com/team/service:v1.2.3",
		},
		{
			name:         "project is prefixed when image is unqualified",
			registryHost: "registry.example.com",
			project:      "team",
			image:        "service",
			tag:          "",
			want:         "registry.example.com/team/service:latest",
		},
		{
			name:         "project is not duplicated when already present",
			registryHost: "registry.example.com",
			project:      "team",
			image:        "team/service",
			tag:          "stable",
			want:         "registry.example.com/team/service:stable",
		},
		{
			name:  "no registry host",
			image: "library/nginx",
			tag:   "alpine",
			want:  "library/nginx:alpine",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PullReference(tc.registryHost, tc.project, tc.image, tc.tag); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestPullCommand(t *testing.T) {
	got := PullCommand("https://registry.example.com", "team", "service", "v1")
	want := "docker pull registry.example.com/team/service:v1"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
