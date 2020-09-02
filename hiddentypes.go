package hiddentypes

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"strings"

	"go/token"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "hiddentypes is ..."

// Analyzer is ...
var Analyzer = &analysis.Analyzer{
	Name: "hiddentypes",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
		buildssa.Analyzer,
	},
}

var flagType string
var flagFuncs string

type TypeName struct {
	Pkg  string
	Name string
}

type FuncName struct {
	Pkg  string
	Recv *string
	Name string
}

func init() {
	Analyzer.Flags.StringVar(&flagType, "type", "", "target type")
	Analyzer.Flags.StringVar(&flagFuncs, "funcs", "", "target functions")
}

func run(pass *analysis.Pass) (interface{}, error) {
	fmt.Printf("RUN")

	targetTypeName, targetFuncNames, err := setParams(flagType, flagFuncs)

	log.Printf("targetTypeName: %v", targetTypeName)
	log.Printf("targetFuncNames: %v", targetFuncNames)

	if err != nil {
		return nil, err
	}

	if targetTypeName.Name == "" || len(targetFuncNames) == 0 {
		return nil, nil
	}

	tt := analysisutil.ObjectOf(pass, targetTypeName.Pkg, targetTypeName.Name).Type()
	//log.Printf("tt: %v", tt)

	fs := collectTargetFuncs(pass, targetFuncNames)
	//log.Printf("fs: %v", fs)

	candidate := filterCallInstrPos(pass, fs)
	//log.Printf("insts: %v", candidate)
	//for c, _ := range candidate {
	//	log.Printf("candidate: %v [%v]", c, pass.Fset.Position(c))
	//}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		callexpr := n.(*ast.CallExpr)
		//log.Printf("CHALLENGE: %v [%v]", callexpr.Pos(), pass.Fset.Position(callexpr.Pos()))
		//log.Printf("CHALLENGE(LP): %v [%v]", callexpr.Lparen, pass.Fset.Position(callexpr.Lparen))
		if candidate[callexpr.Lparen] { // dirty hack
			//log.Printf("FOUND: %v", callexpr)
			for _, arg := range callexpr.Args {
				typ := pass.TypesInfo.TypeOf(arg)
				if types.Identical(typ, tt) {
					pass.Reportf(callexpr.Pos(), "NG")
					break
				}
			}
		}
	})
	return nil, nil
}

func setParams(typename, funcnames string) (TypeName, []FuncName, error) {
	var targetTypeName TypeName
	var targetFuncNames = []FuncName{}

	tn := strings.Split(strings.TrimSpace(typename), ".")
	if len(tn) != 2 {
		log.Fatalf("invalid flag (type): %v", tn)
		return TypeName{}, []FuncName{}, errors.New("invalid flag (type)")
	}
	targetTypeName.Pkg = tn[0]
	targetTypeName.Name = tn[1]

	for _, fn := range strings.Fields(funcnames) {
		f := strings.Split(fn, ".")

		// package function : pkg.func
		if len(f) == 2 {
			targetFuncNames = append(targetFuncNames, FuncName{f[0], nil, f[1]})
			continue
		}

		// method : (pkg.recv).func or (*pkg.recv).func
		if len(f) == 3 {
			pkg := strings.TrimLeft(f[0], "(")
			recv := strings.TrimLeft(f[1], ")")
			if pkg != "" && pkg[0] == '*' {
				pkg = pkg[1:]
				recv = "*" + recv
			}
			targetFuncNames = append(targetFuncNames, FuncName{pkg, &recv, f[2]})
			continue
		}

		log.Fatalf("invalid flag (funcs): %v", f)
		return TypeName{}, []FuncName{}, errors.New("invalid flag (funcs)")
	}

	return targetTypeName, targetFuncNames, nil
}

func collectTargetFuncs(pass *analysis.Pass, target []FuncName) []*types.Func {
	fs := []*types.Func{}

	// targetとして渡された関数のtypes.Object，types.Typeを取得
	for _, fn := range target {
		if fn.Recv == nil {
			// get function
			f := analysisutil.ObjectOf(pass, fn.Pkg, fn.Name).(*types.Func)
			if f != nil {
				fs = append(fs, f)
			}
		} else {
			// get method
			t := analysisutil.TypeOf(pass, fn.Pkg, *fn.Recv)
			if t == nil {
				continue
			}
			m := analysisutil.MethodOf(t, fn.Name)
			if m != nil {
				fs = append(fs, m)
			}
		}
	}

	// TODO:ネストした関数を取得

	return fs
}

func filterCallInstrPos(pass *analysis.Pass, fs []*types.Func) map[token.Pos]bool {
	result := make(map[token.Pos]bool)
	srcFunc := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	for _, sf := range srcFunc {
		for _, b := range sf.Blocks {
			for _, i := range b.Instrs {
				for _, f := range fs {
					if analysisutil.Called(i, nil, f) {
						result[i.Pos()] = true
					}
				}
			}
		}
	}

	return result
}
