package main

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

type OutputValue struct {
	path string
}

func NewOutputValue() *OutputValue {
	return &OutputValue{}
}

func (o *OutputValue) String() string {
	if len(o.path) > 0 {
		return o.path
	}
	return "(STDOUT)"
}

func (o *OutputValue) Set(value string) error {
	o.path = value
	return nil
}

func (o *OutputValue) Reset() {
	o.path = ""
}

func (o *OutputValue) Type() string {
	return "[path]"
}

func (o *OutputValue) WriterCloser() (io.WriteCloser, error) {
	if len(o.path) > 0 {
		return os.OpenFile(o.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
	}
	return os.Stdout, nil
}

var outputValue = NewOutputValue()

func AddOutputFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(outputValue, "output", "o", "Path to output to, default is to stdout")
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		outputValue.Reset()
	})
}
