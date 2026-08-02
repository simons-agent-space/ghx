package subcmd

import "testing"

func TestValidateAPIPath(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"relative path", "/repos/owner/repo/pulls", false},
		{"path with query", "/repos/owner/repo/pulls?state=open", false},
		{"https api.github.com", "https://api.github.com/repos/owner/repo/pulls", false},
		{"https api.github.com with query", "https://api.github.com/repos/owner/repo/pulls?per_page=100", false},
		{"http rejected", "http://api.github.com/foo", true},
		{"arbitrary https host", "https://attacker.example/x", true},
		{"https localhost", "https://localhost:8080/admin", true},
		{"https 169.254 link-local", "https://169.254.169.254/latest/meta-data/", true},
		{"non-github https", "https://example.com/foo", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAPIPath(tc.path)
			if tc.wantErr && err == nil {
				t.Errorf("path %q: expected error, got nil", tc.path)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("path %q: unexpected error: %v", tc.path, err)
			}
		})
	}
}
