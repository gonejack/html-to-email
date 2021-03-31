package main

import (
	"log"
	"os"

	"github.com/gonejack/html-to-email/cmd"
	"github.com/spf13/cobra"
)

var (
	from    string
	to      string
	verbose bool

	prog = &cobra.Command{
		Use:   "html-to-email *.html",
		Short: "Command line tool for converting html to email.",
		Run: func(c *cobra.Command, args []string) {
			err := run(c, args)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
)

func init() {
	log.SetOutput(os.Stdout)

	prog.Flags().SortFlags = false

	flags := prog.PersistentFlags()
	{
		flags.SortFlags = false
		flags.StringVarP(&from, "from", "f", "", "set From field")
		flags.StringVarP(&to, "to", "t", "", "set To field")
		flags.BoolVarP(&verbose, "verbose", "v", false, "verbose")
	}
}
func run(c *cobra.Command, args []string) error {
	exec := cmd.HTMLToEmail{
		From:    from,
		To:      to,
		Verbose: verbose,
	}

	return exec.Run(args)
}
func main() {
	_ = prog.Execute()
}
