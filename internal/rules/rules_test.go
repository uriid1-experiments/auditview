package rules

import "testing"

// A relative path is accepted by the editor but rejected by auditctl,
// which aborts the load and leaves the kernel without the rules below it.
func TestValidateRejectsRelativePaths(t *testing.T) {
	cases := []struct {
		name string
		line string
		bad  bool
	}{
		{"never with relative path", "-a never,exit -F exe=/usr/bin/systemd-tmpfiles -F path=openvpn -F perm=r", true},
		{"always with relative dir", "-a always,exit -F dir=.ssh -F perm=r -k cred_ssh", true},
		{"watch with relative path", "-w etc/crontab -p wa -k persist_cron", true},
		{"absolute dir", "-a always,exit -F dir=/home/u/.ssh -F perm=r -k cred_ssh", false},
		{"absolute path", "-a never,exit -F exe=/usr/bin/fish -F path=/home/u/.bash_history", false},
		{"absolute watch", "-w /etc/crontab -p wa -k persist_cron", false},
		{"rule without a path", "-a always,exit -F arch=b64 -S execve -k exec", false},
		{"comment", "# -F path=openvpn", false},
		{"control line", "-b 16384", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			problems := Validate(c.line)
			if got := len(problems) > 0; got != c.bad {
				t.Fatalf("Validate(%q) = %v, want problem: %v", c.line, problems, c.bad)
			}
		})
	}
}

// The whole point is catching the bad line before augenrules --load runs,
// so a file must be flagged even when every other line is fine.
func TestValidateReportsLineNumber(t *testing.T) {
	content := "# noise\n-w /etc/crontab -p wa -k persist_cron\n-a never,exit -F path=openvpn -F perm=r\n"
	problems := Validate(content)
	if len(problems) != 1 {
		t.Fatalf("want exactly one problem, got %v", problems)
	}
	if want := "line 3"; problems[0][:len(want)] != want {
		t.Fatalf("problem does not point at line 3: %q", problems[0])
	}
}
