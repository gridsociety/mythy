package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataInfoAndRangeAndInfoVis(t *testing.T) {
	dir := t.TempDir()
	tplPath := filepath.Join(dir, "TEST-VB0-x")
	body := `<?xml version="1.0" encoding="UTF-8"?>
<DEVICE NAME="X" IDENTIFICATION="111" PROTOCOLRELEASE="0100" XMLRELEASE="0101" FAMILY="X">
  <WSDL><definitions name="X"><types><schema/></types>
    <message name="UL1" type="RREG" num="100" dim="2" CLASS="ULONG_RAM"/>
    <message name="UL1Response"/>
    <message name="MB_address" type="WREG" num="200" dim="1" CLASS="UBYTE_PARAM"/>
    <message name="MB_addressResponse"/>
  </definitions></WSDL>
  <MENU>
    <GROUP NAME="Set">
      <DATA NAME="MB_address" DSC="Modbus address" TIPO="UBYTE" VALORE="1" DEFAULT="1">
        <RANGE VALUE="0,247,1" EXT="DEC,,0,-1"/>
      </DATA>
      <DATA NAME="UL1" DSC="L1 voltage" TIPO="ULONG" READONLY="YES">
        <INFO UM="Hz" DP="3" KVIS="1000.000000" AR="1" AS="0" OPER="X"/>
        <INFOVIS SELECT="VisMisura2"/>
      </DATA>
    </GROUP>
  </MENU>
</DEVICE>
`
	if err := os.WriteFile(tplPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tpl, err := ParseTemplate(tplPath)
	if err != nil {
		t.Fatal(err)
	}
	mb := tpl.Menu.FindGroup("Set").FindData("MB_address")
	if mb == nil || mb.Range == nil {
		t.Fatalf("Range missing on MB_address: %+v", mb)
	}
	if mb.Range.Min != 0 || mb.Range.Max != 247 || mb.Range.Step != 1 {
		t.Errorf("MB_address Range = %+v", mb.Range)
	}
	if mb.Range.Format != "DEC" {
		t.Errorf("Format = %q", mb.Range.Format)
	}

	ul := tpl.Menu.FindGroup("Set").FindData("UL1")
	if ul.Info == nil || ul.Info.Unit != "Hz" || ul.Info.Decimals != 3 || ul.Info.Scale != 1000.0 {
		t.Errorf("UL1.Info = %+v", ul.Info)
	}
	if ul.Info.Extra["AR"] != "1" {
		t.Errorf("UL1.Info.Extra[AR] = %q", ul.Info.Extra["AR"])
	}
	if ul.InfoVis == nil || ul.InfoVis.Select != "VisMisura2" {
		t.Errorf("UL1.InfoVis = %+v", ul.InfoVis)
	}
}
