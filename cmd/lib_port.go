package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"go.bug.st/serial"

	"github.com/fornellas/cgs/serialtcp"
)

var portName string
var defaultPortName = ""

var address string
var defaultAddress = ""

var timeout time.Duration
var defaultTimeout = 5 * time.Second

func AddPortFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&portName, "port-name", "p", defaultPortName, "Serial port name to open")
	cmd.PersistentFlags().StringVarP(&address, "address", "a", defaultAddress, "TCP address to connect to")
	cmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", defaultTimeout, "TCP connect timeout")
}

func GetOpenPortFn() (func(context.Context, *serial.Mode) (serial.Port, error), error) {
	if portName != "" && address != "" {
		return nil, fmt.Errorf("flags --port-name and --address can not be set simultaneously")
	}

	if portName != "" {
		return func(ctx context.Context, mode *serial.Mode) (serial.Port, error) {
			return serial.Open(portName, mode)
		}, nil
	}

	if address != "" {
		return func(ctx context.Context, mode *serial.Mode) (serial.Port, error) {
			return serialtcp.TcpPortDial(ctx, address, timeout)
		}, nil
	}

	return nil, fmt.Errorf("either --port-name or --address must be set")
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		portName = defaultPortName
		address = defaultAddress
		timeout = defaultTimeout
		outputValue.Reset()
	})
}
