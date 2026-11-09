package types

import (
	"context"
	"testing"
)

func TestAcademicAssistantBuiltinExists(t *testing.T) {
	if err := LoadBuiltinAgentsConfig("../../config"); err != nil {
		t.Fatal(err)
	}
	a := GetBuiltinAgentWithContext(context.Background(), "academic-assistant", 1)
	if a == nil {
		t.Fatal("没有 academic-assistant")
	}
	found := false
	for _, n := range a.Config.AllowedTools {
		if n == "academic_db" {
			found = true
		}
	}
	if !found {
		t.Fatal("academic_db 不在 allowed_tools 里")
	}
}
