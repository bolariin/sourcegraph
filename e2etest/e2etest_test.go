package e2etest

import (
	"flag"
	"os"
	"testing"
)

func TestDefFlow(t *testing.T) {
	runE2E(t, "def_flow")
}

func TestLoginFlow(t *testing.T) {
	runE2E(t, "login_flow")
}

func TestRegisterFlow(t *testing.T) {
	runE2E(t, "register_flow")
}

func TestRepoFlow(t *testing.T) {
	runE2E(t, "repo_flow")
}

var skipMsg string

func TestMain(m *testing.M) {
	flag.Parse()
	err := parseEnv()
	if err != nil {
		skipMsg = "parseEnv: " + err.Error()
	}
	os.Exit(m.Run())
}

func runE2E(t *testing.T, name string) {
	var test *Test
	for i := range tr.tests {
		if tr.tests[i].Name == name {
			test = tr.tests[i]
		}
	}
	if test == nil {
		t.Fatal("Could not find test")
	}
	if skipMsg != "" {
		t.Skip(skipMsg)
	}
	wd, err := tr.newWebDriver()
	if err != nil {
		t.Skip("newWebDriver:", err)
	}
	defer wd.Quit()
	e2eT := tr.newT(test, wd)
	e2eT.testingT = &seleniumTestingT{t}
	err = test.Func(e2eT)
	if err != nil {
		t.Fatal(err)
	}
}

type seleniumTestingT struct {
	t *testing.T
}

func (t *seleniumTestingT) Fatalf(fmt string, v ...interface{}) {
	t.t.Fatalf(fmt, v...)
}
