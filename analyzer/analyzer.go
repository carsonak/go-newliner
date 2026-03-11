// Package analyzer implements the go-newliner analysis pass.
package analyzer

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "gonewliner",
	Doc:      "enforces blank-line formatting rules after closing braces, declarations, and goroutine statements",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// We need to walk function bodies to inspect statement lists.
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		var body *ast.BlockStmt

		switch n := n.(type) {
		case *ast.FuncDecl:
			body = n.Body
		case *ast.FuncLit:
			body = n.Body
		}

		if body == nil {
			return
		}

		checkStmtList(pass, body.List)
	})

	return nil, nil
}

// checkStmtList inspects a list of statements and applies the three rules.
func checkStmtList(pass *analysis.Pass, stmts []ast.Stmt) {
	for i, stmt := range stmts {
		// Recurse into nested blocks.
		recurseIntoStmt(pass, stmt)

		if i+1 >= len(stmts) {
			continue
		}

		next := stmts[i+1]

		switch s := stmt.(type) {
		case *ast.IfStmt:
			checkClosingBrace(pass, s, stmts, i)
		case *ast.ForStmt:
			checkClosingBraceGeneric(pass, s.Body, next)
		case *ast.RangeStmt:
			checkClosingBraceGeneric(pass, s.Body, next)
		case *ast.SwitchStmt:
			checkClosingBraceGeneric(pass, s.Body, next)
		case *ast.TypeSwitchStmt:
			checkClosingBraceGeneric(pass, s.Body, next)
		case *ast.SelectStmt:
			checkClosingBraceGeneric(pass, s.Body, next)
		case *ast.BlockStmt:
			checkClosingBraceGeneric(pass, s, next)
		case *ast.DeclStmt:
			checkDecl(pass, stmts, i)
		case *ast.AssignStmt:
			if s.Tok == token.DEFINE {
				checkDecl(pass, stmts, i)
			}
		case *ast.GoStmt:
			checkGo(pass, stmts, i)
		}
	}
}

// recurseIntoStmt descends into compound statements to check inner statement lists.
func recurseIntoStmt(pass *analysis.Pass, stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		checkStmtList(pass, s.List)
	case *ast.IfStmt:
		if s.Body != nil {
			checkStmtList(pass, s.Body.List)
		}

		if s.Else != nil {
			recurseIntoStmt(pass, s.Else)
		}
	case *ast.ForStmt:
		if s.Body != nil {
			checkStmtList(pass, s.Body.List)
		}
	case *ast.RangeStmt:
		if s.Body != nil {
			checkStmtList(pass, s.Body.List)
		}
	case *ast.SwitchStmt:
		if s.Body != nil {
			for _, c := range s.Body.List {
				if cc, ok := c.(*ast.CaseClause); ok {
					checkStmtList(pass, cc.Body)
				}
			}
		}
	case *ast.TypeSwitchStmt:
		if s.Body != nil {
			for _, c := range s.Body.List {
				if cc, ok := c.(*ast.CaseClause); ok {
					checkStmtList(pass, cc.Body)
				}
			}
		}
	case *ast.SelectStmt:
		if s.Body != nil {
			for _, c := range s.Body.List {
				if cc, ok := c.(*ast.CommClause); ok {
					checkStmtList(pass, cc.Body)
				}
			}
		}
	}
}

// ---------- Rule 1: Closing curly braces ----------

// checkClosingBrace handles *ast.IfStmt specifically for the defer-cleanup exception.
func checkClosingBrace(pass *analysis.Pass, ifStmt *ast.IfStmt, stmts []ast.Stmt, idx int) {
	if idx+1 >= len(stmts) {
		return
	}

	next := stmts[idx+1]
	block := outermostIfBody(ifStmt)

	// Exception B: next token is }, ] or ).
	if nextNonWSIsClosing(pass, block.Rbrace) {
		return
	}

	gap := lineGap(pass, block.Rbrace, next.Pos())

	// Exception A: defer cleanup pattern.
	if gap == 1 && idx >= 1 {
		if names := lhsNames(stmts[idx-1]); isNilCheck(ifStmt, names) {
			if deferStmt, ok := next.(*ast.DeferStmt); ok {
				if matchesDeferCleanup(stmts[idx-1], deferStmt) {
					return
				}
			}
		}
	}

	if gap < 2 {
		reportMissingBlank(pass, block.Rbrace, "closing brace should be followed by a blank line")
	}
}

// outermostIfBody returns the block of the deepest else branch, i.e. the final closing brace.
func outermostIfBody(ifStmt *ast.IfStmt) *ast.BlockStmt {
	for ifStmt.Else != nil {
		if elseIf, ok := ifStmt.Else.(*ast.IfStmt); ok {
			ifStmt = elseIf
		} else if elseBlock, ok := ifStmt.Else.(*ast.BlockStmt); ok {
			return elseBlock
		} else {
			break
		}
	}

	return ifStmt.Body
}

// checkClosingBraceGeneric applies Rule 1 to non-if block statements.
func checkClosingBraceGeneric(pass *analysis.Pass, block *ast.BlockStmt, next ast.Stmt) {
	if block == nil {
		return
	}

	// Exception B: next token is }, ] or ).
	if nextNonWSIsClosing(pass, block.Rbrace) {
		return
	}

	gap := lineGap(pass, block.Rbrace, next.Pos())

	if gap < 2 {
		reportMissingBlank(pass, block.Rbrace, "closing brace should be followed by a blank line")
	}
}

// ---------- Rule 2: Declarations ----------

func checkDecl(pass *analysis.Pass, stmts []ast.Stmt, idx int) {
	// Find the end of the contiguous block of declaration statements.
	end := idx

	for end+1 < len(stmts) {
		if isDeclLike(stmts[end+1]) {
			end++
		} else {
			break
		}
	}
	// Only check at the last decl in the contiguous block.
	if idx != end {
		return
	}

	if end+1 >= len(stmts) {
		return
	}

	next := stmts[end+1]

	// Exception A: next is an if that checks a variable from this declaration block against nil.
	if ifStmt, ok := next.(*ast.IfStmt); ok {
		names := collectDeclBlockNames(stmts, idx, end)

		if isNilCheck(ifStmt, names) {
			return
		}
	}

	gap := lineGap(pass, stmts[end].End(), next.Pos())

	if gap < 2 {
		reportMissingBlank(pass, stmts[end].End(), "declaration should be followed by a blank line")
	}
}

// isDeclLike returns true for *ast.DeclStmt and short variable declarations (:=).
func isDeclLike(stmt ast.Stmt) bool {
	if _, ok := stmt.(*ast.DeclStmt); ok {
		return true
	}

	if assign, ok := stmt.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
		return true
	}

	return false
}

// ---------- Rule 3: Goroutines ----------

func checkGo(pass *analysis.Pass, stmts []ast.Stmt, idx int) {
	// Find the end of the contiguous block of GoStmts.
	end := idx

	for end+1 < len(stmts) {
		if _, ok := stmts[end+1].(*ast.GoStmt); ok {
			end++
		} else {
			break
		}
	}

	if idx != end {
		return
	}

	if end+1 >= len(stmts) {
		return
	}

	next := stmts[end+1]

	// Exception A: next non-whitespace char is }.
	if nextNonWSIsClosing(pass, stmts[end].End()) {
		return
	}

	gap := lineGap(pass, stmts[end].End(), next.Pos())

	if gap < 2 {
		reportMissingBlank(pass, stmts[end].End(), "go statement should be followed by a blank line")
	}
}

// ---------- Helpers ----------

// lineGap returns the number of lines between fromPos (exclusive) and toPos (inclusive).
// A gap of 2 means there is exactly one blank line between them.
func lineGap(pass *analysis.Pass, from, to token.Pos) int {
	fromLine := pass.Fset.Position(from).Line
	toLine := pass.Fset.Position(to).Line

	return toLine - fromLine
}

// nextNonWSIsClosing returns true if the first non-whitespace byte after pos in the
// source file is }, ] or ).
func nextNonWSIsClosing(pass *analysis.Pass, pos token.Pos) bool {
	tokFile := pass.Fset.File(pos)

	if tokFile == nil {
		return false
	}

	offset := tokFile.Offset(pos)
	// pos points at the character itself (e.g. '}'), so start scanning after it.
	offset++

	for _, f := range pass.Files {
		fPos := pass.Fset.Position(f.Pos())
		fEnd := pass.Fset.Position(f.End())

		if pass.Fset.Position(pos).Filename != fPos.Filename {
			continue
		}
		// Read from the token.File size.
		src := readSource(pass, f)

		if src == nil {
			return false
		}

		_ = fEnd // keep linter quiet
		for offset < len(src) {
			ch := src[offset]

			if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
				offset++
				continue
			}

			return ch == '}' || ch == ']' || ch == ')'
		}

		return false
	}

	return false
}

// readSource retrieves the raw source bytes for a file via the token.File.
func readSource(pass *analysis.Pass, file *ast.File) []byte {
	tokFile := pass.Fset.File(file.Pos())

	if tokFile == nil {
		return nil
	}

	size := tokFile.Size()
	start := tokFile.Pos(0)
	end := tokFile.Pos(size)
	// Use pass.ReadFile if available (go1.22+), otherwise fall back to the
	// beginning/end range trick.  Since we need raw bytes and ReadFile
	// isn't available in all versions, we use the file-set offset approach.
	_ = end
	// We actually iterate over pass.OtherFiles or rely on the fact that
	// analysis framework gives us source.  In practice we can grab the source
	// through os.ReadFile using the filename.
	return readFileBytes(pass.Fset.Position(start).Filename)
}

// isNilCheck returns true if the if statement's condition is of the form
// `<name> != nil` where <name> is one of the provided variable names.
func isNilCheck(ifStmt *ast.IfStmt, names map[string]bool) bool {
	if len(names) == 0 {
		return false
	}

	cond, ok := ifStmt.Cond.(*ast.BinaryExpr)

	if !ok {
		return false
	}

	if cond.Op != token.NEQ {
		return false
	}

	xIdent, xOk := cond.X.(*ast.Ident)
	yIdent, yOk := cond.Y.(*ast.Ident)

	// <name> != nil
	if xOk && names[xIdent.Name] && yOk && yIdent.Name == "nil" {
		return true
	}

	// nil != <name>
	if xOk && xIdent.Name == "nil" && yOk && names[yIdent.Name] {
		return true
	}

	return false
}

// lhsNames returns the set of names on the left-hand side of an assignment or
// short variable declaration. Returns nil for other statement types.
func lhsNames(stmt ast.Stmt) map[string]bool {
	assign, ok := stmt.(*ast.AssignStmt)

	if !ok {
		return nil
	}

	names := make(map[string]bool)

	for _, expr := range assign.Lhs {
		if ident, ok := expr.(*ast.Ident); ok {
			names[ident.Name] = true
		}
	}

	return names
}

// collectDeclBlockNames collects all LHS variable names from a contiguous
// block of declaration statements (stmts[start] through stmts[end]).
func collectDeclBlockNames(stmts []ast.Stmt, start, end int) map[string]bool {
	names := make(map[string]bool)

	for i := start; i <= end; i++ {
		switch s := stmts[i].(type) {
		case *ast.AssignStmt:
			for _, expr := range s.Lhs {
				if ident, ok := expr.(*ast.Ident); ok {
					names[ident.Name] = true
				}
			}
		case *ast.DeclStmt:
			if genDecl, ok := s.Decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, ident := range vs.Names {
							names[ident.Name] = true
						}
					}
				}
			}
		}
	}

	return names
}

// matchesDeferCleanup checks whether a defer statement is cleaning up a
// resource that was assigned right before the error check.
//
// Pattern:
//
//	x, err := something()
//	if err != nil { return err }
//	defer x.Close()
//
// We check if any identifier on the LHS of the assignment preceding the if
// matches the receiver or argument of the deferred call.
func matchesDeferCleanup(preceding ast.Stmt, deferStmt *ast.DeferStmt) bool {
	assignStmt, ok := preceding.(*ast.AssignStmt)

	if !ok {
		return false
	}

	lhsNames := make(map[string]bool)

	for _, expr := range assignStmt.Lhs {
		if ident, ok := expr.(*ast.Ident); ok {
			lhsNames[ident.Name] = true
		}
	}

	call := deferStmt.Call

	if call == nil {
		return false
	}

	// Check selector receiver: defer x.Close()
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			if lhsNames[ident.Name] {
				return true
			}
		}
	}

	// Check call arguments: defer close(x)
	for _, arg := range call.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			if lhsNames[ident.Name] {
				return true
			}
		}
	}

	return false
}

// reportMissingBlank emits a diagnostic with a SuggestedFix that inserts a newline.
func reportMissingBlank(pass *analysis.Pass, pos token.Pos, message string) {
	// We want to insert a \n right after the end of the current line.
	// The insert position is the beginning of the next line.
	insertPos := beginningOfNextLine(pass, pos)

	pass.Report(analysis.Diagnostic{
		Pos:     pos,
		Message: message,
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message: "insert blank line",
				TextEdits: []analysis.TextEdit{
					{
						Pos:     insertPos,
						End:     insertPos,
						NewText: []byte("\n"),
					},
				},
			},
		},
	})
}

// beginningOfNextLine returns the token.Pos at the start of the line
// following the line that contains pos.
func beginningOfNextLine(pass *analysis.Pass, pos token.Pos) token.Pos {
	tokFile := pass.Fset.File(pos)

	if tokFile == nil {
		return pos
	}

	position := pass.Fset.Position(pos)
	line := position.Line
	// If there is a next line in the file, return its start.
	if line < tokFile.LineCount() {
		return tokFile.LineStart(line + 1)
	}
	// Otherwise, return pos at end of file.
	return token.Pos(tokFile.Base() + tokFile.Size())
}
