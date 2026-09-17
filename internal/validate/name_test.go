package validate

import "testing"

func TestTunnelName_Valid(t *testing.T) {
	valid := []string{
		"grafana",
		"portainer",
		"home-assistant",
		"a",
		"a1",
		"pr-142",
		"x123456789012345678901234567890123456789012345678901234567890", // 63 chars
	}
	for _, name := range valid {
		if err := TunnelName(name); err != nil {
			t.Errorf("TunnelName(%q) returned error, want nil: %v", name, err)
		}
	}
}

func TestTunnelName_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"../foo",
		"../../foo",
		"foo/../../bar",
		`foo\..\bar`,
		"foo/bar",
		`foo\bar`,
		"-foo",
		"Foo",
		"FOO",
		"foo bar",
		"foo\nbar",
		"foo.bar",
		"foo_bar",
		"foo$(bar)",
		"foo;bar",
		"x1234567890123456789012345678901234567890123456789012345678901234", // 67 chars, too long
	}
	for _, name := range invalid {
		if err := TunnelName(name); err == nil {
			t.Errorf("TunnelName(%q) returned nil, want error", name)
		}
	}
}
