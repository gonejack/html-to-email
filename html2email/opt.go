package html2email

import (
	"path/filepath"

	"github.com/alecthomas/kong"
)

type Options struct {
	From    string `short:"f" help:"Set From field."`
	To      string `short:"t" help:"Set To field."`
	Verbose bool   `short:"v" help:"Verbose printing."`
	About   bool   `help:"About."`

	HTML []string `arg:"" optional:""`
}

func MustParseOption() (opt Options) {
	kong.Parse(&opt,
		kong.Name("html-to-email"),
		kong.Description("This command line converts .html file to .eml"),
		kong.UsageOnError(),
	)
	if len(opt.HTML) == 0 || opt.HTML[0] == "*.html" {
		opt.HTML, _ = filepath.Glob("*.html")
	}
	return
}
