package processor

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

type patch struct {
	pos   token.Pos
	stmts []ast.Stmt
}

func patchFile(fset *token.FileSet, file *ast.File, patches ...patch) error {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return err
	}
	src := buf.Bytes()

	offset := int(file.FileStart) - 1
	for _, patch := range patches {
		buf.Reset()

		buf.WriteString("\n")
		if err := format.Node(&buf, fset, patch.stmts); err != nil {
			return err
		}
		buf.WriteString("\n")

		pos := int(patch.pos) - offset
		src = append(src[:pos], append(buf.Bytes(), src[pos:]...)...)
		// patch positions after need to be shifted up relative to updates in src by buffer
		offset -= buf.Len()
	}

	nfile, err := parser.ParseFile(fset, fset.Position(file.Pos()).Filename, src, parser.ParseComments)
	if err != nil {
		return err
	}
	*file = *nfile
	return nil
}
