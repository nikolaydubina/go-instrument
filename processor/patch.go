package processor

import (
	"bytes"
	"fmt"
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

			buf.WriteString(fmt.Sprintf("\n/*line %s:%d:%d*/", filename, firstStmtPos.Line, firstStmtPos.Column))
		}

		pos := int(patch.pos) - offset
		src = append(src[:pos], append(buf.Bytes(), src[pos:]...)...)
		// patch positions after need to be shifted up relative to updates in src by buffer
		offset -= buf.Len()
	}

	// Post-process the source to ensure line directives are immediately before statements
	// This removes any whitespace between /*line*/ directives and the following statements
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
	lines := bytes.Split(src, []byte("\n"))
	var newLines [][]byte
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Check if this line contains a /*line*/ directive
		if bytes.Contains(line, []byte("/*line ")) && bytes.Contains(line, []byte("*/")) {
			// This line has a line directive
			// Find the next non-empty, non-comment line
			var nextContentLine []byte
			var nextContentIndex int = -1
			
			for j := i + 1; j < len(lines); j++ {
				nextLine := lines[j]
				trimmed := bytes.TrimSpace(nextLine)
				
				// Skip empty lines and comment-only lines
				if len(trimmed) == 0 || bytes.HasPrefix(trimmed, []byte("//")) {
					continue
				}
				
				// Found the next content line
				nextContentLine = nextLine
				nextContentIndex = j
				break
			}
			
			if nextContentIndex != -1 {
				// Combine the directive with the content line
				trimmed := bytes.TrimLeft(nextContentLine, " \t")
				combinedLine := append(line, trimmed...)
				newLines = append(newLines, combinedLine)
				
				// Skip all lines up to and including the content line
				i = nextContentIndex
			} else {
				// No content found, just add the directive line
				newLines = append(newLines, line)
			}
		} else {
			newLines = append(newLines, line)
		}
	}
	
	return bytes.Join(newLines, []byte("\n"))
}

func formatNodeToBytes(fset *token.FileSet, node any) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
