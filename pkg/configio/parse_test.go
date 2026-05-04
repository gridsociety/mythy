package configio_test

import (
	"errors"
	"testing"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/configio"
)

func loadTpl(t *testing.T) *catalog.Template {
	t.Helper()
	tpl, err := catalog.ParseTemplate("../../testdata/us/TEST-VB0-a")
	if err != nil {
		t.Fatal(err)
	}
	return tpl
}

func TestParseValidYAML(t *testing.T) {
	in := []byte(`
mythy_version: 1
device: { product: TEST-VX0-a }
settings:
  MB_address: 5
  MB_baudrate: "9600 baud"
`)
	cf, err := configio.Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cf.Settings["MB_address"] != int(5) && cf.Settings["MB_address"] != int64(5) {
		t.Errorf("MB_address = %v", cf.Settings["MB_address"])
	}
}

func TestValidateUnknownKey(t *testing.T) {
	tpl := loadTpl(t)
	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings:     map[string]any{"NotARealName": 1},
	}
	err := configio.Validate(cf, tpl, "TEST-VX0-a")
	var unk *configio.UnknownKeyError
	if !errors.As(err, &unk) {
		t.Errorf("expected UnknownKeyError, got %T %v", err, err)
	}
}

func TestValidateProductMismatch(t *testing.T) {
	tpl := loadTpl(t)
	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "PROX-VX0-e"},
		Settings:     map[string]any{},
	}
	err := configio.Validate(cf, tpl, "TEST-VX0-a")
	var pm *configio.ProductMismatchError
	if !errors.As(err, &pm) {
		t.Errorf("expected ProductMismatchError, got %T %v", err, err)
	}
}

func TestValidateBadEnum(t *testing.T) {
	tpl := loadTpl(t)
	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings:     map[string]any{"MB_baudrate": "totally-bogus"},
	}
	err := configio.Validate(cf, tpl, "TEST-VX0-a")
	if err == nil {
		t.Error("expected error for invalid enum label")
	}
}

func TestValidateGoodValuesPass(t *testing.T) {
	tpl := loadTpl(t)
	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings: map[string]any{
			"MB_address":  int(5),
			"MB_baudrate": "9600 baud",
		},
	}
	if err := configio.Validate(cf, tpl, "TEST-VX0-a"); err != nil {
		t.Fatalf("expected pass; got %v", err)
	}
}
