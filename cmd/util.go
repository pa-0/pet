package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/knqyf263/pet/config"
	"github.com/knqyf263/pet/dialog"
	"github.com/knqyf263/pet/snippet"
)

func editFile(command, file string, startingLine int) error {
	// Note that this works for most unix editors (nano, vi, vim, etc)
	// TODO: Remove for other kinds of editors - this is only for UX
	command += " +" + strconv.Itoa(startingLine) + " " + file
	return run(command, os.Stdin, os.Stdout)
}

func filter(options []string, tag string) (commands []string, err error) {
	var snippets snippet.Snippets
	if err := snippets.Load(); err != nil {
		return commands, fmt.Errorf("load snippet failed: %v", err)
	}

	if 0 < len(tag) {
		var filteredSnippets snippet.Snippets
		for _, snippet := range snippets.Snippets {
			for _, t := range snippet.Tag {
				if tag == t {
					filteredSnippets.Snippets = append(filteredSnippets.Snippets, snippet)
				}
			}
		}
		snippets = filteredSnippets
	}

	snippetTexts := map[string]snippet.SnippetInfo{}
	var text string
	for _, s := range snippets.Snippets {
		command := s.Command
		if strings.ContainsAny(command, "\n") {
			command = strings.Replace(command, "\n", "\\n", -1)
		}
		t := fmt.Sprintf("[%s]: %s", s.Description, command)

		tags := ""
		for _, tag := range s.Tag {
			tags += fmt.Sprintf(" #%s", tag)
		}
		t += tags

		snippetTexts[t] = s
		if config.Flag.Color || config.Conf.General.Color {
			t = fmt.Sprintf("[%s]: %s%s",
				color.HiRedString(s.Description), command, color.HiCyanString(tags))
		}
		text += t + "\n"
	}

	var buf bytes.Buffer
	selectCmd := fmt.Sprintf("%s %s",
		config.Conf.General.SelectCmd, strings.Join(options, " "))
	err = run(selectCmd, strings.NewReader(text), &buf)
	if err != nil {
		return nil, nil
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	var params [][2]string

	// If only one line is selected, search for params in the command
	if len(lines) == 1 {
		snippetInfo := snippetTexts[lines[0]]
		params = dialog.SearchForParams(snippetInfo.Command)
	} else {
		params = nil
	}

	if params != nil {
		snippetInfo := snippetTexts[lines[0]]
		dialog.CurrentCommand = snippetInfo.Command
		dialog.GenerateParamsLayout(params, dialog.CurrentCommand)
		res := []string{dialog.FinalCommand}
		return res, nil
	}
	for _, line := range lines {
		snippetInfo := snippetTexts[line]
		commands = append(commands, fmt.Sprint(snippetInfo.Command))
	}
	return commands, nil
}

// CountLines returns the number of lines in a certain buffer
func CountLines(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
