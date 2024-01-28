package processor

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

type patch struct {
	pos   int
	stmts []ast.Stmt
}

func (p *Processor) patchFile(fset *token.FileSet, file *ast.File, patches ...patch) error {
	src, err := formatNode(fset, file)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	offset := -int(file.FileStart) + 1
	for _, patch := range patches {
		buf.Reset()

		buf.WriteString("\n")
		if err := format.Node(&buf, fset, patch.stmts); err != nil {
			return err
		}
		buf.WriteString("\n")

		pos := patch.pos + offset
		src = append(src[:pos], append(buf.Bytes(), src[pos:]...)...)
		offset += buf.Len()
	}

	p.updateFile(fset, file, src)

	return nil
}

func formatNode(fset *token.FileSet, node any) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (p *Processor) updateFile(fset *token.FileSet, file *ast.File, newSrc []byte) error {
	var err error

	if newSrc == nil {
		newSrc, err = formatNode(fset, file)
	}

	fname := fset.Position(file.Pos()).Filename

	nfile, err := parser.ParseFile(fset, fname, newSrc, parser.ParseComments)
	if err != nil {
		return err
	}

	*file = *nfile

	return nil
}
