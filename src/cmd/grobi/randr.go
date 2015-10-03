package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Output encapsulates a physical output with detected modes.
type Output struct {
	Name      string
	Modes     []Mode
	Connected bool
}

// Mode is an output mode that may be active or default.
type Mode struct {
	Name    string
	Default bool
	Active  bool
}

var (
	// ErrNotModeLine is returned by ParseModeLine when the line doesn't match
	// the format for a mode line.
	ErrNotModeLine = errors.New("not a mode line")
)

// ParseOutputLine returns the output parsed from the string.
func ParseOutputLine(line string) (Output, error) {
	output := Output{}

	ws := bufio.NewScanner(bytes.NewReader([]byte(line)))
	ws.Split(bufio.ScanWords)

	if !ws.Scan() {
		return Output{}, fmt.Errorf("line too short, name not found: %s", line)
	}
	output.Name = ws.Text()

	if !ws.Scan() {
		return Output{}, fmt.Errorf("line too short, state not found: %s", line)
	}

	switch ws.Text() {
	case "connected":
		output.Connected = true
	case "disconnected":
		output.Connected = false
	default:
		return Output{}, fmt.Errorf("unknown state %q", ws.Text())
	}

	return output, nil
}

// ParseModeLine returns the mode parsed from the string.
func ParseModeLine(line string) (mode Mode, err error) {
	if !strings.HasPrefix(line, "  ") {
		return Mode{}, ErrNotModeLine
	}

	ws := bufio.NewScanner(bytes.NewReader([]byte(line)))
	ws.Split(bufio.ScanWords)

	if !ws.Scan() {
		return Mode{}, fmt.Errorf("line too short, mode name not found: %s", line)
	}
	mode.Name = ws.Text()

	if !ws.Scan() {
		return Mode{}, fmt.Errorf("line too short, no refresh rate found: %s", line)
	}
	rate := ws.Text()

	if rate[len(rate)-1] == '+' {
		mode.Default = true
	}

	if rate[len(rate)-2] == '*' {
		mode.Active = true
	}

	// handle single-word "+", which happens when a mode is default but not active
	if ws.Scan() && ws.Text() == "+" {
		mode.Default = true
	}

	return mode, nil
}

// RandrParse returns the list of outputs parsed from the reader.
func RandrParse(rd io.Reader) (outputs []Output, err error) {
	ls := bufio.NewScanner(rd)

	const (
		StateStart = iota
		StateOutput
		StateMode
	)

	var (
		state  = StateStart
		output Output
	)

nextLine:
	for ls.Scan() {
		line := ls.Text()

		for {
			switch state {
			case StateStart:
				if strings.HasPrefix(line, "Screen ") {
					state = StateOutput
					continue nextLine
				}
				return nil, fmt.Errorf(`first line should start with "Screen", found: %v`, line)

			case StateOutput:
				output, err = ParseOutputLine(line)
				if err != nil {
					return nil, err
				}
				state = StateMode
				continue nextLine

			case StateMode:
				mode, err := ParseModeLine(line)
				if err == ErrNotModeLine {
					outputs = append(outputs, output)
					output = Output{}
					state = StateOutput
					continue
				}

				if err != nil {
					return nil, err
				}

				output.Modes = append(output.Modes, mode)
				continue nextLine
			}
		}
	}

	if output.Name != "" {
		outputs = append(outputs, output)
	}

	return outputs, nil
}