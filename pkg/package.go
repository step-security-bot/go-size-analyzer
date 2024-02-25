package pkg

import (
	"debug/gosym"
	"github.com/goretk/gore"
	"log"
	"maps"
)

func (k *KnownInfo) loadPackages(file *gore.GoFile) (*TypedPackages, error) {
	log.Println("Loading packages...")

	pkgs := new(TypedPackages)

	pkgs.NameToPkg = make(map[string]*Package)

	pclntab, err := file.PCLNTab()
	if err != nil {
		return nil, err
	}

	self, err := file.GetPackages()
	if err != nil {
		return nil, err
	}
	selfPkgs, n, err := loadGorePackages(self, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Self = selfPkgs
	maps.Copy(pkgs.NameToPkg, n)

	grStd, _ := file.GetSTDLib()
	std, n, err := loadGorePackages(grStd, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Std = std
	maps.Copy(pkgs.NameToPkg, n)

	grVendor, _ := file.GetVendors()
	vendor, n, err := loadGorePackages(grVendor, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Vendor = vendor
	maps.Copy(pkgs.NameToPkg, n)

	grGenerated, _ := file.GetGeneratedPackages()
	generated, n, err := loadGorePackages(grGenerated, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Generated = generated
	maps.Copy(pkgs.NameToPkg, n)

	grUnknown, _ := file.GetUnknown()
	unknown, n, err := loadGorePackages(grUnknown, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Unknown = unknown
	maps.Copy(pkgs.NameToPkg, n)

	log.Println("Loading packages done")

	return pkgs, nil
}

func loadGorePackages(gr []*gore.Package, k *KnownInfo, pclntab *gosym.Table) ([]*Package, map[string]*Package, error) {
	pkgs := make([]*Package, 0, len(gr))
	nameToPkg := make(map[string]*Package)
	for _, g := range gr {
		pkg, err := loadGorePackage(g, k, pclntab)
		if err != nil {
			return nil, nil, err
		}
		pkgs = append(pkgs, pkg)
		nameToPkg[pkg.Name] = pkg
	}
	return pkgs, nameToPkg, nil
}

func loadGorePackage(pkg *gore.Package, k *KnownInfo, pclntab *gosym.Table) (*Package, error) {
	ret := &Package{
		Name:  pkg.Name,
		Addrs: make([]*Addr, 0),
		grPkg: pkg,
	}

	files := map[string]*File{}

	getFile := func(path string) *File {
		f, ok := files[path]
		if !ok {
			nf := &File{
				Path:      path,
				Functions: make([]*gore.Function, 0),
				Methods:   make([]*gore.Method, 0),
			}
			files[path] = nf
			return nf
		}
		return f
	}

	setAddrMark := func(addr, size uint64, meta GoPclntabMeta) {
		k.FoundAddr.Insert(addr, size, ret, AddrPassGoPclntab, meta)
	}

	for _, m := range pkg.Methods {
		src, _, _ := pclntab.PCToLine(m.Offset)

		setAddrMark(m.Offset, m.End-m.Offset, GoPclntabMeta{
			FuncName:    Deduplicate(m.Name),
			PackageName: Deduplicate(m.PackageName),
			Type:        FuncTypeMethod,
			Receiver:    Deduplicate(m.Receiver),
		})

		mf := getFile(src)
		mf.Methods = append(mf.Methods, m)
	}

	for _, f := range pkg.Functions {
		src, _, _ := pclntab.PCToLine(f.Offset)

		setAddrMark(f.Offset, f.End-f.Offset, GoPclntabMeta{
			FuncName:    Deduplicate(f.Name),
			PackageName: Deduplicate(f.PackageName),
			Type:        FuncTypeFunction,
			Receiver:    Deduplicate(""),
		})

		ff := getFile(src)
		ff.Functions = append(ff.Functions, f)
	}

	filesSlice := make([]*File, 0, len(files))
	for _, f := range files {
		filesSlice = append(filesSlice, f)
	}

	ret.Files = filesSlice

	return ret, nil
}

type TypedPackages struct {
	Self      []*Package
	Std       []*Package
	Vendor    []*Package
	Generated []*Package
	Unknown   []*Package

	NameToPkg map[string]*Package // available after loadPackages, loadPackagesFromGorePackage[s]
}

func (tp *TypedPackages) GetPackages() []*Package {
	var pkgs []*Package

	// use this order to make it more likely to set the disasm result to the right package
	pkgs = append(pkgs, tp.Unknown...)
	pkgs = append(pkgs, tp.Generated...)
	pkgs = append(pkgs, tp.Std...)
	pkgs = append(pkgs, tp.Vendor...)
	pkgs = append(pkgs, tp.Self...)
	return pkgs
}

func (tp *TypedPackages) GetPackageAndCountFn() ([]*Package, int) {
	pkgs := tp.GetPackages()
	cnt := 0
	for _, p := range pkgs {
		cnt += len(p.GetFunctions())
		cnt += len(p.GetMethods())
	}
	return pkgs, cnt
}

type Package struct {
	Name  string
	Files []*File
	Addrs []*Addr // from symbols and disasm result
	grPkg *gore.Package
}

func (p *Package) GetFunctions() []*gore.Function {
	var fns []*gore.Function
	for _, f := range p.Files {
		fns = append(fns, f.Functions...)
	}
	return fns
}

func (p *Package) GetMethods() []*gore.Method {
	var mds []*gore.Method
	for _, f := range p.Files {
		mds = append(mds, f.Methods...)
	}
	return mds
}