package compiler

import (
	"sort"

	"github.com/zombocoder/goboot/model"
)

// collectDeclarations surfaces every annotated declaration on the model so
// plugin Analyzers and Generators can act on their own annotations (§46.5). The
// result is ordered deterministically (package, file, line, name) because plugin
// generators must produce deterministic output.
func collectDeclarations(scan *ScanResult, app *model.Application) {
	for _, decl := range scan.Declarations {
		if len(decl.Annotations) == 0 {
			continue
		}
		recv := ""
		if decl.Recv != nil {
			recv = decl.Recv.Name()
		}
		app.Declarations = append(app.Declarations, model.AnnotatedDecl{
			Package:     decl.PkgPath,
			Name:        decl.Name,
			Receiver:    recv,
			Target:      decl.Target,
			Annotations: decl.Annotations,
			Position:    decl.Pos,
		})
	}
	sort.SliceStable(app.Declarations, func(i, j int) bool {
		di, dj := app.Declarations[i], app.Declarations[j]
		if di.Package != dj.Package {
			return di.Package < dj.Package
		}
		if di.Position.Filename != dj.Position.Filename {
			return di.Position.Filename < dj.Position.Filename
		}
		if di.Position.Line != dj.Position.Line {
			return di.Position.Line < dj.Position.Line
		}
		return di.Name < dj.Name
	})
}
