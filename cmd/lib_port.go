package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.bug.st/serial"

	"github.com/fornellas/cgs/serialtcp"
)

var portName string
var defaultPortName = ""

var address string
var defaultAddress = ""

func AddPortFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&portName, "port-name", "p", defaultPortName, "Serial port name to open")
	cmd.PersistentFlags().StringVarP(&address, "address", "a", defaultAddress, "TCP address to connect to")
}

func GetOpenPortFn() (func(*serial.Mode) (serial.Port, error), error) {
	if portName != "" && address != "" {
		return nil, fmt.Errorf("flags --port-name and --address can be set simultaneously")
	}

	if portName != "" {
		return func(mode *serial.Mode) (serial.Port, error) {
			return serial.Open(portName, mode)
		}, nil
	}

	if address != "" {
		return func(mode *serial.Mode) (serial.Port, error) {
			return serialtcp.TcpPortDial("tcp", address)
		}, nil
	}

	return nil, fmt.Errorf("either --port-name or --address can be set")
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		portName = defaultPortName
		address = defaultAddress
		outputValue.Reset()
	})
}
