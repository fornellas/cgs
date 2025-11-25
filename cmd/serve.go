package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

var listenAddress string
var defaultListenAddress = "127.0.0.1:9999"

func handleServeConnection(ctx context.Context, conn net.Conn, port string) error {
	logger := log.MustLogger(ctx)

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetNoDelay(true); err != nil {
			return fmt.Errorf("failed to set TCP no delay: %w", err)
		}
	}

	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	logger.Info("Opening serial port")
	serialPort, err := serial.Open(port, mode)
	if err != nil {
		return fmt.Errorf("failed to open: %s: %w", port, err)
	}

	errCh := make(chan error, 2)

	logger.Info("Copying I/O")
	go func() {
		_, err := io.Copy(conn, serialPort)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(serialPort, conn)
		errCh <- err
	}()

	err = <-errCh
	logger.Info("Closing connection")
	err = errors.Join(err, conn.Close())
	logger.Info("Closing port")
	err = errors.Join(err, serialPort.Close())
	logger.Info("Waiting for copy routine to return")
	err = errors.Join(err, <-errCh)

	return err
}

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a TCP server connected to a serial port.",
	Long:  "Opens serial port and a TCP server, and pipes communication between both. There's NO security implemented, this can only be used in secure networks at your own risk.",
	Args:  cobra.NoArgs,
	Run: GetRunFn(func(cmd *cobra.Command, args []string) error {
		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"port-name", portName,
			"listen-address", listenAddress,
		)
		cmd.SetContext(ctx)

		logger.Info("Listening")
		listener, err := net.Listen("tcp", listenAddress)
		if err != nil {
			return fmt.Errorf("failed to listen: %s: %w", listenAddress, err)
		}
		defer func() { errors.Join(err, listener.Close()) }()

		for {
			logger.Info("Accepting connection")
			conn, err := listener.Accept()
			if err != nil {
				logger.Error("Failed to accept connection", "error", err)
				continue
			}
			connCtx, connLogger := log.MustWithGroupAttrs(
				ctx,
				"Connection",
				"LocalAddr", conn.LocalAddr(),
				"RemoteAddr", conn.RemoteAddr(),
			)
			connLogger.Info("Accepted")

			if err := handleServeConnection(connCtx, conn, portName); err != nil {
				connLogger.Error("Failed to handle connection", "error", err)
			}
		}
	}),
}

func init() {
	ServeCmd.PersistentFlags().StringVarP(&portName, "port-name", "p", defaultPortName, "Serial port name to open")
	if err := ServeCmd.MarkPersistentFlagRequired("port-name"); err != nil {
		panic(err)
	}
	ServeCmd.PersistentFlags().StringVar(&listenAddress, "listen-address", defaultListenAddress, "TCP address to listen on (host:port)")

	RootCmd.AddCommand(ServeCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		portName = defaultPortName
		listenAddress = defaultListenAddress
	})
}
