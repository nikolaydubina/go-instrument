package processor

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
)

type patch struct {
	pos   token.Pos
	stmts []ast.Stmt
}

func patchFile(fset *token.FileSet, file *ast.File, patches ...patch) error {
	// Patches must be applied in the ascending order, otherwise the
	// modified source file will become corrupted.
	sort.Slice(patches, func(i, j int) bool { return patches[i].pos < patches[j].pos })

	src, err := formatNodeToBytes(fset, file)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	offset := int(file.FileStart) - 1
	for _, patch := range patches {
		buf.Reset()

		buf.WriteRune('\n')
		if err := format.Node(&buf, fset, patch.stmts); err != nil {
			return err
		}
		buf.WriteRune('\n')

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

func formatNodeToBytes(fset *token.FileSet, node any) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
