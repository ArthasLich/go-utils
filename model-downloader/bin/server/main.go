package main

import (
	"go-utils/model-downloader/logic"
	"log"
	"path/filepath"

	"github.com/spf13/pflag"
)

var (
	flagDB       = pflag.StringP("database", "d", "models.db", "Database file")
	flagSavePath = pflag.StringP("save-path", "s", "", "dir to save model files")
	flagHelp     = pflag.BoolP("help", "h", false, "show help")
	flagPython   = pflag.String("python", "python3.10", "set python cmd path whilch used to query model size")
	flagPort     = pflag.Uint16P("port", "p", 9986, "set listen port")
)

func main() {
	pflag.Parse()
	if *flagHelp {
		pflag.Usage()
		return
	}

	if len(*flagSavePath) == 0 {
		log.Fatalf("save path must be set")
	}
	// 获取保存路径的绝对路径
	savePath, err := filepath.Abs(*flagSavePath)
	if err != nil {
		log.Fatalf("get absolute path of save path failed: %v", err)
	}
	savePath, err = filepath.EvalSymlinks(savePath)
	if err != nil {
		log.Fatalf("evaluate symlinks of save path failed: %v", err)
	}
	
	logic.Init(*flagDB, savePath, *flagPython, *flagPort)
	logic.Run()
}
