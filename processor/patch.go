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
	pos    token.Pos
	stmts  []ast.Stmt
	fnBody *ast.BlockStmt
}

func patchFile(fset *token.FileSet, file *ast.File, patches ...patch) error {
	// patches must be applied in the ascending order, otherwise the modified source file will become corrupted.
	sort.Slice(patches, func(i, j int) bool { return patches[i].pos < patches[j].pos })

	src, err := formatNodeToBytes(fset, file)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	offset := int(file.FileStart) - 1
	for _, patch := range patches {
		buf.Reset()

		if len(patch.stmts) > 0 {
			buf.WriteRune('\n')
			if err := format.Node(&buf, fset, patch.stmts); err != nil {
				return err
			}
		}

		// line directives to preserve line numbers of functions (for accurate panic stack traces)
		// https://github.com/golang/go/blob/master/src/cmd/compile/doc.go#L171
		if patch.fnBody != nil && len(patch.fnBody.List) > 0 && len(patch.stmts) > 0 {
			buf.WriteString("\n/*line ")
			buf.WriteString(fset.Position(patch.fnBody.List[0].Pos()).String())
			buf.WriteString("*/\n")
		}

		pos := int(patch.pos) - offset
		src = append(src[:pos], append(buf.Bytes(), src[pos:]...)...)
		// patch positions after need to be shifted up relative to updates in src by buffer
		offset -= buf.Len()
	}

	// post-process the source to ensure line directives are immediately before statements
	src = cleanupLineDirectives(src)

	nfile, err := parser.ParseFile(fset, fset.Position(file.Pos()).Filename, src, parser.ParseComments)
	if err != nil {
		return err
	}

	*file = *nfile
	return nil
}

// cleanupLineDirectives removes whitespace between /*line*/ directives and following statements
func cleanupLineDirectives(src []byte) []byte {
	var buf bytes.Buffer
	lines := bytes.Split(src, []byte("\n"))

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		trimmed := bytes.TrimSpace(line)

		if isLineDirective := bytes.HasPrefix(trimmed, []byte("/*line ")) && bytes.HasSuffix(trimmed, []byte("*/")); isLineDirective {
			// This line has only a line directive
			// Find the next non-empty, non-comment line
			var nextContentLine []byte
			var nextContentIndex = -1

			for j := i + 1; j < len(lines); j++ {
				nextLine := lines[j]
				nextTrimmed := bytes.TrimSpace(nextLine)

				// Skip empty lines and comment-only lines that start with //
				if len(nextTrimmed) == 0 || bytes.HasPrefix(nextTrimmed, []byte("//")) {
					continue
				}

				// Found the next content line
				nextContentLine = nextLine
				nextContentIndex = j
				break
			}

			if nextContentIndex != -1 {
				contentTrimmed := bytes.TrimLeft(nextContentLine, " \t")
				indentation := nextContentLine[:len(nextContentLine)-len(contentTrimmed)]

				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.Write(indentation)
				buf.Write(trimmed)
				buf.Write(contentTrimmed)

				i = nextContentIndex
			} else {
				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.Write(line)
			}
		} else {
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.Write(line)
		}
	}

	return buf.Bytes()
}

func formatNodeToBytes(fset *token.FileSet, node any) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
