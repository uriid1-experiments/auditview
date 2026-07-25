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
		{"never с относительным path", "-a never,exit -F exe=/usr/bin/systemd-tmpfiles -F path=openvpn -F perm=r", true},
		{"always с относительным dir", "-a always,exit -F dir=.ssh -F perm=r -k cred_ssh", true},
		{"watch с относительным путём", "-w etc/crontab -p wa -k persist_cron", true},
		{"абсолютный dir", "-a always,exit -F dir=/home/u/.ssh -F perm=r -k cred_ssh", false},
		{"абсолютный path", "-a never,exit -F exe=/usr/bin/fish -F path=/home/u/.bash_history", false},
		{"абсолютный watch", "-w /etc/crontab -p wa -k persist_cron", false},
		{"правило без пути", "-a always,exit -F arch=b64 -S execve -k exec", false},
		{"комментарий", "# -F path=openvpn", false},
		{"control-строка", "-b 16384", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			problems := Validate(c.line)
			if got := len(problems) > 0; got != c.bad {
				t.Fatalf("Validate(%q) = %v, ожидалось наличие проблемы: %v", c.line, problems, c.bad)
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
		t.Fatalf("ожидалась одна проблема, получено %v", problems)
	}
	if want := "строка 3"; problems[0][:len(want)] != want {
		t.Fatalf("проблема не указывает на строку 3: %q", problems[0])
	}
}
