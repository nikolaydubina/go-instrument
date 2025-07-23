package processor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
)

type patch struct {
	pos      token.Pos
	stmts    []ast.Stmt
	filename string
	fnBody   *ast.BlockStmt
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

		// Insert the instrumentation statements
		buf.WriteRune('\n')
		if err := format.Node(&buf, fset, patch.stmts); err != nil {
			return err
		}
		
		// Add line directive to preserve original line numbers for the first original statement
		if patch.fnBody != nil && patch.filename != "" && len(patch.fnBody.List) > 0 {
			firstStmt := patch.fnBody.List[0]
			firstStmtPos := fset.Position(firstStmt.Pos())
			basename := filepath.Base(patch.filename)
			// We want the first statement after the blank line to appear at its original line number
			// Account for the blank line after the //line directive
			buf.WriteString(fmt.Sprintf("\n//line %s:%d", basename, firstStmtPos.Line-3))
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
