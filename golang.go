package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type golangApplier struct {
}

func (g *golangApplier) CheckHeader(target *os.File, t *TagContext) (bool, error) {

	// Check compiler flags.
	cFlags, cbufs, err := g.checkSpecialConditions(target)
	if err != nil {
		return false, err
	}
	target.Seek(0, 0)

	if cFlags == AutoGenerated {
		return true, nil
	}

	tbuf, err := os.ReadFile(filepath.Join(t.templatePath, "go.txt"))
	if err != nil {
		return false, err
	}

	var templateBuf string
	if cFlags == CompilerFlags {
		templateBuf = strings.Join(cbufs, "\n") + "\n"
		templateBuf += "\n"
		templateBuf += string(tbuf)
	} else {
		templateBuf = string(tbuf)
	}

	targetBuf := make([]byte, len(templateBuf))

	n, err := target.Read(targetBuf)
	if err != nil {
		return false, err
	}

	if n == len(templateBuf) {
		if strings.Compare(templateBuf, string(targetBuf)) == 0 {
			return true, nil
		}
	}

	return false, nil
}

func (g *golangApplier) ApplyHeader(path string, t *TagContext) error {

	file, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	t.templateFiles.goTemplateFile.Seek(0, 0)

	headerExist, err := g.CheckHeader(file, t)
	if err != nil {
		return err
	}

	if headerExist {
		return nil
	}

	// Reset the read pointers to begining of file.
	t.templateFiles.goTemplateFile.Seek(0, 0)
	file.Seek(0, 0)

	sFlags, flags, err := g.checkSpecialConditions(file)
	if err != nil {
		return err
	}
	file.Seek(0, 0)

	tempFile := path + ".tmp"
	tFile, err := os.OpenFile(tempFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer tFile.Close()

	reader := bufio.NewReader(file)
	if sFlags == CompilerFlags {
		for _, f := range flags {
			tFile.WriteString(f + "\n")
			_, _, err = reader.ReadLine()
		}
		tFile.WriteString("\n")
		_, _, err = reader.ReadLine()
	}

	if sFlags == AutoGenerated {
		// This should not hit.
		panic(err)
	}

	_, err = io.Copy(tFile, t.templateFiles.goTemplateFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(tFile, reader)
	if err != nil {
		return err
	}

	err = os.Rename(tempFile, path)
	if err != nil {
		return err
	}
	return nil
}

func (g *golangApplier) checkSpecialConditions(target *os.File) (uint8, []string, error) {
	reader := bufio.NewReader(target)
	buf, _, err := reader.ReadLine()
	if err != nil {
		return NormalFiles, nil, err
	}
	line := string(buf)

	if strings.HasPrefix(line, "//") && strings.Contains(line, "DO NOT EDIT") {
		return AutoGenerated, nil, nil
	}

	var buildTags []string

	// Go 1.17 compiler flags (e.g., `//go:build !windows`), or old compiler
	// flags (e.g., `// +build !windows`).
	// checks for Package comments as per https://blog.golang.org/godoc-documenting-go-code
	if strings.HasPrefix(line, "//go:") || strings.HasPrefix(line, "// +build") {
		buildTags = append(buildTags, line)

		// read next line too: (`// +build !windows`)
		for {
			buf2, _, err := reader.ReadLine()
			if err != nil {
				break
			}
			if strings.HasPrefix(string(buf2), "// +build ") {
				buildTags = append(buildTags, string(buf2))
			} else {
				break
			}
		}
		return CompilerFlags, buildTags, nil
	}

	return NormalFiles, nil, nil
}
