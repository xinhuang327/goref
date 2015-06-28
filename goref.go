package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

const (
	CMD_mvc_goto_view   = "mvc_goto_view"
	CMD_mvc_goto_action = "mvc_goto_action"
)

var inputFilePath string
var inputOffset int
var inputCmd string

func init() {
	flag.IntVar(&inputOffset, "inputOffset", 0, "inputOffset")
	flag.StringVar(&inputFilePath, "inputFilePath", "", "inputFilePath")
	flag.StringVar(&inputCmd, "inputCmd", "", "inputCmd")
	flag.Parse()
}

func main() {
	if inputFilePath == "" || inputCmd == "" {
		flag.Usage()
		return
	}

	input := InputInfo{
		InputFilePath: inputFilePath,
		InputOffset:   inputOffset,
		InputCmd:      inputCmd,
	}

	result := NewRedirectAction()
	result.Do(input)

	jsonStr, err := EncodeToJSON(ResposneInfo{
		InputInfo:  input,
		ResultInfo: result,
	})
	if err != nil {
		fmt.Errorf("EncodeToJSON error %s", err)
	}
	fmt.Println(jsonStr)
}

type InputInfo struct {
	InputFilePath string
	InputOffset   int
	InputCmd      string
}

type ResposneInfo struct {
	InputInfo  InputInfo
	ResultInfo interface{}
}

type RedirectAction struct {
	FilePath   string
	Offset     int
	ResultType string
	DebugInfo  map[string]interface{}
}

func NewRedirectAction() *RedirectAction {
	return &RedirectAction{
		ResultType: "RedirectAction",
		DebugInfo:  make(map[string]interface{}),
	}
}

func (r *RedirectAction) Do(input InputInfo) error {
	switch input.InputCmd {
	case CMD_mvc_goto_view:
		var funcName string
		var funcRecvTypeName string
		var fset = token.NewFileSet()
		node, err := parser.ParseFile(fset, input.InputFilePath, nil, parser.DeclarationErrors)
		if err != nil {
			return err
		}
		ast.Inspect(node, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				start := fset.Position(n.Pos())
				end := fset.Position(n.End())
				if isBetween(input.InputOffset, start.Offset, end.Offset) {
					funcName = x.Name.Name
					r.DebugInfo["funcName"] = funcName
					switch recvType := x.Recv.List[0].Type.(type) {
					case *ast.StarExpr:
						funcRecvTypeName = recvType.X.(*ast.Ident).Name
					case *ast.Ident:
						funcRecvTypeName = recvType.Name
					}
					funcRecvTypeName = strings.TrimSuffix(funcRecvTypeName, "Controller")
					r.DebugInfo["funcRecvTypeName"] = funcRecvTypeName
					return false
				}
			}
			return funcName == ""
		})
		if funcRecvTypeName != "" && funcName != "" {
			viewFilePath := filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(input.InputFilePath)), "web/templates"), funcRecvTypeName+"/"+funcName+".html")
			r.FilePath = viewFilePath
		}
	case CMD_mvc_goto_action:
		templatesDir := filepath.Dir(filepath.Dir(input.InputFilePath))
		r.DebugInfo["templatesDir"] = templatesDir
		if filepath.Base(templatesDir) == "templates" {
			actionName := strings.TrimSuffix(filepath.Base(input.InputFilePath), ".html")
			ctrlName := filepath.Base(filepath.Dir(input.InputFilePath))
			r.DebugInfo["actionName"] = actionName
			r.DebugInfo["ctrlName"] = ctrlName
			ctrlFilePath := filepath.Join(filepath.Dir(filepath.Dir(templatesDir)), "ctrls/"+ctrlName+".go")
			r.FilePath = ctrlFilePath
		}
	}
	return nil
}

func EncodeToJSON(obj interface{}) (jsonString string, err error) {
	var jsonData []byte
	if jsonData, err = json.Marshal(obj); err == nil {
		jsonString = string(jsonData)
	}
	return
}

func isBetween(n, start, end int) bool {
	return (n >= start && n <= end)
}
