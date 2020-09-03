package hiddentypes

import (
	"errors"
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
	log.Printf("RUN")

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
	log.Printf("fs: %v", fs)

	ws := collectWrappedTargetFuncs(pass, fs) // TODO: 調べるべき引数番号も取得する
	log.Printf("ws: %v", ws)

	candidate := filterCallInstrPos(pass, fs)
	//log.Printf("insts: %v", candidate)
	//for c, _ := range candidate {
	//	log.Printf("candidate: %v [%v]", c, pass.Fset.Position(c))
	//}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspect.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
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

	// target関数の取得
	// target関数は全ての引数が検査対象
	// targetとして渡された関数のtypes.Object，types.Typeを取得
	for _, fn := range target {
		if fn.Recv == nil {
			// get function
			f, _ := analysisutil.ObjectOf(pass, fn.Pkg, fn.Name).(*types.Func)
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

	// 関数Fがtarget関数を内部で呼び出している && 関数Xの引数が一つでも変更されずにtarget関数へ渡されている
	// 上の条件を満たす関数もtarget関数として扱う
	// 関数Xに対する検査はtarget関数へ渡されている引数についてのみでよくて，かつその引数の型がターゲットの型か何かしらのインターフェース型であるときで十分
	// この条件のもとで，不動点を探す？
	// 「どの引数について検査すべきなのか」は別フェーズで判断？

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

// target関数をラップしているような関数もtargetとする
// 内部でtarget関数を呼び出す関数&&引数をそのままtarget関数に渡している&&その引数がtarget型or何かしらのインターフェース型
// だったら追加する
func collectWrappedTargetFuncs(pass *analysis.Pass, fs []*types.Func) []*types.Func {
	result := []*types.Func{}
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspect.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		fdecl := n.(*ast.FuncDecl)
		//log.Printf("collectWrap: %v", fdecl)
		argpos := isCall(pass, fdecl, fs)
		//log.Printf("collectWrap:argpos: %v", argpos)
		if len(argpos) == 0 {
			return
		}
		// TODO: 検査すべき引数番号の情報も返す
		f := pass.TypesInfo.ObjectOf(fdecl.Name).(*types.Func)
		result = append(result, f)
	})

	return result
}

// ある関数(fdecl)がfsのどれかを呼び出しているか検査する
// 仮引数を直接関数呼び出しに渡しているような仮引数のインデックスを返す
func isCall(pass *analysis.Pass, fdecl *ast.FuncDecl, fs []*types.Func) map[int]bool {
	result := make(map[int]bool)
	ast.Inspect(fdecl, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			//log.Printf("isCall:n: %v", n)
			var fn *types.Func
			switch f := n.Fun.(type) {
			case *ast.Ident:
				fn, _ = pass.TypesInfo.ObjectOf(f).(*types.Func)
				if fn == nil {
					return true
				}
			case *ast.SelectorExpr:
				fn, _ = pass.TypesInfo.ObjectOf(f.Sel).(*types.Func)
				if fn == nil {
					return true
				}
			default:
				return true
			}

			objmap := argsObjMap(pass, fdecl)
			//log.Printf("isCall:objmap: %v", objmap)

			for _, target := range fs {
				if fn == target { // ==で比較可能？
					// target関数の呼び出し
					// 実引数が変更されずに使われているか確認(fdeclの仮引数と一致するかで近似)
					for _, arg := range n.Args {
						id, _ := arg.(*ast.Ident)
						if id == nil {
							continue
						}
						obj := pass.TypesInfo.ObjectOf(id)
						pos, ok := objmap[obj]
						if !ok {
							continue
						}

						result[pos] = true
					}
				}
			}
		}
		return true
	})

	return result
}

func argsObjMap(pass *analysis.Pass, fdecl *ast.FuncDecl) map[types.Object]int {
	result := make(map[types.Object]int)
	count := 0
	//log.Printf("argsObjMap: %v", fdecl)
	for _, arg := range fdecl.Type.Params.List {
		for _, name := range arg.Names {
			// TODO: mapに追加するのはtarget型か何かしらのinterface型を持つようなものだけで良い
			//log.Printf("argsObjMap:name: %v", name)
			result[pass.TypesInfo.ObjectOf(name)] = count
			count++
		}
	}
	return result
}
