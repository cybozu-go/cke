package cmd

import (
	"testing"

	"github.com/cybozu-go/cke"
)

func TestKubernetesIssueDefaults(t *testing.T) {
	// CKE itself issues certificates for cke.RoleAdmin.  Should this command
	// ever default to the same name again, audit logs would no longer be able
	// to tell a human operator from CKE's own reconciliation.
	if cke.DefaultUserName == cke.RoleAdmin {
		t.Errorf("the default user name must differ from CKE's own %q", cke.RoleAdmin)
	}
	if got := kubernetesIssueCmd.Flags().Lookup("user").DefValue; got != cke.DefaultUserName {
		t.Errorf("--user: expected %q, actual %q", cke.DefaultUserName, got)
	}

	// Permissions of the issued certificate come from this organization rather
	// than from the user name, so renaming the user must not disturb it.
	if got := kubernetesIssueCmd.Flags().Lookup("group").DefValue; got != cke.AdminGroup {
		t.Errorf("--group: expected %q, actual %q", cke.AdminGroup, got)
	}
}
