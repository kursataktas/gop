package goprj

import (
	"errors"
	"syscall"

	"github.com/qiniu/qlang/modutil"
	"github.com/qiniu/x/log"
)

// -----------------------------------------------------------------------------

// Package represents a set of source files collectively building a Go package.
type Package struct {
	prj *Project
	mod modutil.Module
	dir string

	names PkgNames

	syms map[string]Symbol // cached
	src  *GoPackage
	busy bool
}

func openPackage(dir string, prj *Project) (pkg *Package, err error) {
	mod, err := modutil.LoadModule(dir)
	if err != nil {
		return
	}
	return &Package{prj: prj, mod: mod, dir: dir, names: NewPkgNames()}, nil
}

// Source returns the Go package instance.
func (p *Package) Source() *GoPackage {
	if err := p.requireSource(); err != nil {
		log.Fatal("requireSource failed:", err)
	}
	return p.src
}

// Project returns the project instance.
func (p *Package) Project() *Project {
	return p.prj
}

// ThisModule returns the module instance.
func (p *Package) ThisModule() modutil.Module {
	return p.mod
}

// LookupPkgName lookups a package name by specified PkgPath.
func (p *Package) LookupPkgName(pkgPath string) string {
	return p.names.LookupPkgName(p.mod, pkgPath)
}

// LoadPackage loads a package.
func (p *Package) LoadPackage(pkgPath string) (pkg *Package, err error) {
	if pkgPath == "C" {
		panic("LoadPackage: \"C\" not allow")
	}
	pi, err := p.mod.Lookup(pkgPath)
	if err != nil {
		return
	}
	return p.prj.OpenPackage(pi.Location)
}

// FindPackageType returns the type of symbol pkgPath.name.
func (p *Package) FindPackageType(pkgPath string, name string) (typ Type, err error) {
	if pkgPath == "C" {
		return p.prj.UniqueType(&NamedType{VersionPkgPath: "C", Name: name}), nil
	}
	pkg, err := p.LoadPackage(pkgPath)
	if err != nil {
		return
	}
	return pkg.FindType(name)
}

// FindType returns the type of symbol.
func (p *Package) FindType(name string) (typ Type, err error) {
	sym, err := p.FindSymbol(name)
	if err != nil {
		return
	}
	tsym, ok := sym.(*TypeSym)
	if !ok {
		return nil, ErrSymbolIsNotAType
	}
	if tsym.Alias {
		return tsym.Type, nil
	}
	return p.prj.UniqueType(&NamedType{VersionPkgPath: p.mod.VersionPkgPath(), Name: name}), nil
}

// FindSymbol lookups symbol info.
func (p *Package) FindSymbol(name string) (sym Symbol, err error) {
	if err = p.requireSource(); err != nil {
		log.Fatal("LookupSymbol failed:", err)
		return
	}
	sym, ok := p.syms[name]
	if ok {
		return
	}
	if p.busy {
		return nil, ErrPackageIsLoading
	}
	return nil, ErrNotFound
}

var (
	// ErrPackageIsLoading error.
	ErrPackageIsLoading = errors.New("package is loading")
	// ErrNotFound error.
	ErrNotFound = syscall.ENOENT
)

// -----------------------------------------------------------------------------
