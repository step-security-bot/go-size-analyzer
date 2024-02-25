package pkg

import "go4.org/intern"

func Deduplicate(s string) string {
	return intern.GetByString(s).Get().(string)
}

type FuncType string

const (
	FuncTypeFunction FuncType = "function"
	FuncTypeMethod   FuncType = "method"
)

type GoPclntabMeta struct {
	FuncName    string
	PackageName string
	Type        FuncType
	Receiver    string // for method only
}

type SymbolMeta struct {
	SymbolName  string
	PackageName string
}

type DisasmMeta struct {
	Source       GoPclntabMeta // disasm all functions from pclntab, so it has all info from pclntab
	DisasmIndex  int
	DisasmString string
}

type AddrParseType int

const (
	AddrPassGoPclntab AddrParseType = iota
	AddrPassSymbol
	AddrPassDisasm
)

func (p AddrParseType) String() string {
	switch p {
	case AddrPassGoPclntab:
		return "pclntab"
	case AddrPassSymbol:
		return "symbol"
	case AddrPassDisasm:
		return "disasm"
	default:
		return "unknown"
	}
}