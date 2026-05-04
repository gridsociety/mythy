package configio_test

import (
	"testing"

	"github.com/gridsociety/mythy/pkg/configio"
	"gopkg.in/yaml.v3"
)

func TestConfigFileMarshalRoundTrip(t *testing.T) {
	in := `mythy_version: 1
device:
  product: PROX-VX0-e
  identification: 4660
  serial_number: 100000
  fw_release: "01.00"
  protocol_release: "00.00"
  exported_from: 192.0.2.10
  exported_at: "2026-04-30T14:23:11+02:00"
settings:
  MB_address: 1
  NomeLinea: "SAMPLE-IED"
`
	var f configio.ConfigFile
	if err := yaml.Unmarshal([]byte(in), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if f.MythyVersion != 1 {
		t.Errorf("MythyVersion = %d", f.MythyVersion)
	}
	if f.Device.Product != "PROX-VX0-e" {
		t.Errorf("Product = %q", f.Device.Product)
	}
	if f.Device.Identification != 4660 {
		t.Errorf("Identification = %d", f.Device.Identification)
	}
	if v, ok := f.Settings["MB_address"]; !ok || v != 1 {
		t.Errorf("MB_address = %v", v)
	}
	if v, ok := f.Settings["NomeLinea"]; !ok || v != "SAMPLE-IED" {
		t.Errorf("NomeLinea = %v", v)
	}
}

func TestConfigFileVersionRequired(t *testing.T) {
	in := `mythy_version: 99
device:
  product: PROX-VX0-e
settings: {}
`
	var f configio.ConfigFile
	if err := yaml.Unmarshal([]byte(in), &f); err != nil {
		t.Fatal(err)
	}
	if err := f.Check(); err == nil {
		t.Error("Check() must reject unsupported version")
	}
}
