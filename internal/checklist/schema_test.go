package checklist

import (
	"os"
	"strings"
	"testing"
)

func TestOfficialSchemaAcceptsValidAndRejectsBadStatus(t *testing.T) {
	valid := []byte(`{"title":"synthetic.cklb","id":"123e4567-e89b-12d3-a456-426614174000","stigs":[{"stig_name":"Cisco IOS STIG","display_name":"Cisco IOS","stig_id":"Cisco_IOS_STIG","release_info":"V1R2","uuid":"123e4567-e89b-12d3-a456-426614174001","size":1,"rules":[{"group_id":"V-42","rule_id":"SV-42r1_rule","severity":"high","status":"open","comments":"observed"}]}]}`)
	if err := ValidateSchema(valid); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(valid), `"status":"open"`, `"status":"made_up"`, 1)
	if err := ValidateSchema([]byte(bad)); err == nil {
		t.Fatal("schema allowed invalid status")
	}
	if err := ValidateSchema([]byte(`{"stigs":[]}`)); err == nil {
		t.Fatal("schema allowed missing required fields")
	}
	invalid := strings.Replace(string(valid), `"title":"synthetic.cklb"`, `"title":"synthetic.cklb","unexpected":"field"`, 1)
	p := t.TempDir() + "/one.cklb"
	os.WriteFile(p, []byte(invalid), 0600)
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err = Preflight(f); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatal("preflight allowed schema-invalid checklist", err)
	}
}
