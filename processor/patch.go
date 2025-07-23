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
	"strings"
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
		
		if patch.stmts != nil {
			// Insert the instrumentation statements
			buf.WriteRune('\n')
			if err := format.Node(&buf, fset, patch.stmts); err != nil {
				return err
			}
		}

		// line directive to preserve original line numbers for the first original statement
		// https://github.com/golang/go/blob/master/src/cmd/compile/doc.go#L171
		if patch.fnBody != nil && len(patch.fnBody.List) > 0 {
			// Use the function body position, not the first statement position
			// This gives us the line immediately after the opening brace
			fnBodyPos := fset.Position(patch.fnBody.Pos())
			filename := fset.Position(file.Pos()).Filename
			basename := filepath.Base(filename)
			
			// If this looks like an instrumented file (ends with _instrumented.go), 
			// use the original filename for line directives
			if strings.HasSuffix(basename, "_instrumented.go") {
				basename = strings.TrimSuffix(basename, "_instrumented.go") + ".go"
			}
			
			// The first line in the function body should be fnBodyPos.Line + 1
			// So the line directive should specify that line number
			firstLineNum := fnBodyPos.Line + 1
			buf.WriteString(fmt.Sprintf("\n//line %s:%d", basename, firstLineNum))
		}

		if buf.Len() > 0 {
			pos := int(patch.pos) - offset
			src = append(src[:pos], append(buf.Bytes(), src[pos:]...)...)
			// patch positions after need to be shifted up relative to updates in src by buffer
			offset -= buf.Len()
		}
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
