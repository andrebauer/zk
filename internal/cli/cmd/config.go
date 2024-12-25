package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/zk-org/zk/internal/cli"
	"github.com/zk-org/zk/internal/util/errors"
	"github.com/zk-org/zk/internal/util/strings"
)

// Config lists configuration setting
type Config struct {
	Alias AliasList `cmd group:"cmd" default:"withargs" help:"List all the aliases."`
}

// AliasList lists all the aliases.
type AliasList struct {
	Format     string `group:format short:f placeholder:TEMPLATE   help:"Pretty print the list using a custom template or one of the predefined formats: short, full, json, jsonl."`
	Header     string `group:format                                help:"Arbitrary text printed at the start of the list."`
	Footer     string `group:format default:\n                     help:"Arbitrary text printed at the end of the list."`
	Delimiter  string "group:format short:d default:\n             help:\"Print tags delimited by the given separator.\""
	Delimiter0 bool   "group:format short:0 name:delimiter0        help:\"Print tags delimited by ASCII NUL characters. This is useful when used in conjunction with `xargs -0`.\""
	NoPager    bool   `group:format short:P help:"Do not pipe output into a pager."`
	Quiet      bool   `group:format short:q help:"Do not print the total number of tags found."`
}

func (cmd *AliasList) Run(container *cli.Container) error {
	cmd.Header = strings.ExpandWhitespaceLiterals(cmd.Header)
	cmd.Footer = strings.ExpandWhitespaceLiterals(cmd.Footer)
	cmd.Delimiter = strings.ExpandWhitespaceLiterals(cmd.Delimiter)

	if cmd.Delimiter0 {
		if cmd.Delimiter != "\n" {
			return errors.New("--delimiter and --delimiter0 can't be used together")
		}
		if cmd.Header != "" {
			return errors.New("--footer and --delimiter0 can't be used together")
		}
		if cmd.Footer != "\n" {
			return errors.New("--footer and --delimiter0 can't be used together")
		}

		cmd.Delimiter = "\x00"
		cmd.Footer = "\x00"
	}

	if cmd.Format == "json" || cmd.Format == "jsonl" {
		if cmd.Header != "" {
			return errors.New("--header can't be used with JSON format")
		}
		if cmd.Footer != "\n" {
			return errors.New("--footer can't be used with JSON format")
		}
		if cmd.Delimiter != "\n" {
			return errors.New("--delimiter can't be used with JSON format")
		}

		switch cmd.Format {
		case "json":
			cmd.Delimiter = ","
			cmd.Header = "["
			cmd.Footer = "]\n"

		case "jsonl":
			// > The last character in the file may be a line separator, and it
			// > will be treated the same as if there was no line separator
			// > present.
			// > https://jsonlines.org/
			cmd.Footer = "\n"
		}
	}

	var aliases = container.Config.Aliases
	count := len(aliases)
	keys := make([]string, count)
	i := 0
	for k := range aliases {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	format := cmd.aliasTemplate()

	var err = container.Paginate(cmd.NoPager, func(out io.Writer) error {
		if cmd.Header != "" {
			fmt.Fprint(out, cmd.Header)
		}
		for i, alias := range keys {

			if i > 0 {
				fmt.Fprint(out, cmd.Delimiter)
			}
			if cmd.Format == "" || cmd.Format == "short" {
				fmt.Fprintf(out, format, alias)
			} else {
				fmt.Fprintf(out, format, alias, aliases[alias])
			}

			i += 1
		}
		if cmd.Footer != "" {
			fmt.Fprint(out, cmd.Footer)
		}
		return nil
	})

	if err == nil && !cmd.Quiet {
		if count > 1 {
			fmt.Fprintf(os.Stderr, "\nFound %d %s\n", count, "aliases")
		} else {
			fmt.Fprintf(os.Stderr, "\nFound %d %s\n", count, "alias")
		}
	}
	return err
}

func (cmd *AliasList) aliasTemplate() string {
	format := cmd.Format
	if format == "" {
		format = "short"
	}

	templ, ok := defaultAliasFormats[format]
	if !ok {
		templ = strings.ExpandWhitespaceLiterals(format)
	}

	return templ
}

var defaultAliasFormats = map[string]string{
	"json":  `{"%s":"%s"}`,
	"jsonl": `{"%s":"%s"}`,
	"short": `%s`,
	"full":  `%12s    %s`,
}