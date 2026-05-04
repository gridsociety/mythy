package catalog

import (
	"path/filepath"
	"testing"
)

func TestLinkDataAndCommands(t *testing.T) {
	tpl, err := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}

	mb := tpl.Menu.FindGroup("Set/Base").FindData("MB_address")
	if mb.Message == nil {
		t.Fatal("MB_address.Message not linked")
	}
	if mb.Message.Num != 6146 {
		t.Errorf("MB_address.Message.Num = %d, want 6146", mb.Message.Num)
	}
	if mb.Message.FC() != 3 {
		t.Errorf("MB_address.Message.FC = %d, want 3 (WREG)", mb.Message.FC())
	}

	cmd := tpl.Menu.FindGroup("Commands").Commands[0]
	if cmd.Message == nil || cmd.Message.Num != 5201 {
		t.Errorf("MSG_CMD_RESET_DA_PC: %+v", cmd.Message)
	}
}
