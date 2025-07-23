package processor

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
)

type patch struct {
	pos    token.Pos
	stmts  []ast.Stmt
	fnBody *ast.BlockStmt
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

		// line directives to preserve line numbers of functions (for accurate panic stack traces)
		// https://github.com/golang/go/blob/master/src/cmd/compile/doc.go#L171
		if patch.fnBody != nil && len(patch.fnBody.List) > 0 && patch.stmts != nil {
			firstStmt := patch.fnBody.List[0]
			firstStmtPos := fset.Position(firstStmt.Pos())
			filename := fset.Position(file.Pos()).Filename

			buf.WriteString("\n/*line ")
			buf.WriteString(filename)
			buf.WriteString(":")
			buf.WriteString(strconv.Itoa(firstStmtPos.Line))
			buf.WriteString(":")
			buf.WriteString(strconv.Itoa(firstStmtPos.Column))
			buf.WriteString("*/")
		}

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
