package catalog

// ModuleInfo is one entry from <MODULES>/<MODULE>. The `Variabile`
// field names the catalog register that reports whether the module
// is installed (e.g. EnableBoard_PT100). pkg/configio uses this for
// module-gated export filtering (audit B6).
type ModuleInfo struct {
	Name        string
	Variabile   string
	Description string
}

// Class is a compound type definition (<CLASS NAME=… DIM=…>) with an
// ordered list of <VAR> sub-fields. Used by the codec to encode and
// decode CONTATORE / TIMER / RELE / LED / STATES / SOGLIA / GUASTO /
// SGN_IN / INFO_MISURA / INFO_SCHEDA values.
type Class struct {
	Name string
	Dim  int
	Vars []ClassVar
}

// ClassVar is one sub-field inside a Class. For STRING sub-fields,
// `StringLen` is the character count from an inline <RANGE VALUE="N"/>
// child; the wire width is ceil(N/2) registers (SPEC § 2.10). For
// ENUM/ENUM_LONG/ENUM_BYTE sub-fields with inline `<RANGE VALUE OVERRIDE/>`
// children, `InlineEnum` carries the anonymous enum.
type ClassVar struct {
	Name       string
	Tipo       string
	StringLen  int
	InlineEnum *Enum
}

// Typedef aliases a named compound type to its underlying primitive
// TIPO (e.g. <TYPEDEF NAME="ENUM_LED" TIPO="BIT32"/>).
type Typedef struct {
	Name string
	Tipo string
}

// CompoundFieldOverride captures the per-instance metadata that a
// nested <DATA> child of a compound <DATA TIPO="<class>"> can carry.
// Today only Tipo is consumed (codec layout uses it instead of the
// CLASS VAR TIPO for the matching sub-field); richer fields land here
// without ABI churn when callers start needing them.
type CompoundFieldOverride struct {
	Tipo string // effective TIPO for this sub-field, e.g. "ENUM_LONG" overriding the CLASS "ENUM"
}

// Info is the <INFO UM=… DP=… KVIS=… …/> child of a DATA leaf. Carries
// display formatting for measurement renderers.
type Info struct {
	Unit     string  // <INFO UM=>
	Decimals int     // <INFO DP=>
	Scale    float64 // <INFO KVIS=> — raw / Scale = displayed value
	Extra    map[string]string
}

// InfoVis is the <INFOVIS SELECT="<sibling>"/> child — conditional UI
// visibility. v1 surfaces it via `mythy describe`; not used as a gate.
type InfoVis struct {
	Select string
}

// DataRange is the numeric bounds + display-format directive from
// <RANGE VALUE="lo,hi,step" EXT="format,unit,decimals,scale"/> as a
// child of <DATA>. Distinct from `Enum` whose internal `<RANGE>` form
// encodes enum entries.
type DataRange struct {
	Min, Max, Step int64
	Format         string // typically "DEC"
	Unit           string // "Hz", "%", "^C", "deg", "Hz/s", "En", "fn", ...
	Decimals       int
	Scale          int64 // raw / abs(Scale) = displayed
}
