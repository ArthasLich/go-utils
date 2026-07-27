package main

import (
	"go-utils/model-downloader/logic"
	"log"

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
		return
	}

	if len(*flagSavePath) == 0 {
		log.Fatalf("save path must be set")
	}

	logic.Init(*flagDB, *flagSavePath, *flagPython, *flagPort)
	logic.Run()
}
