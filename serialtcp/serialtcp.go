package serialtcp

import (
	"errors"
	"net"
	"time"

	"go.bug.st/serial"
)

// TcpPort partially implements serial.Port interface over a TCP connection.
type TcpPort struct {
	conn        net.Conn
	readTimeout time.Duration
}

func TcpPortDial(network, address string) (*TcpPort, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return &TcpPort{conn: conn}, nil
}

func (tp *TcpPort) SetMode(mode *serial.Mode) error {
	return errors.New("not supported")
}

func (tp *TcpPort) Read(p []byte) (n int, err error) {
	deadline := time.Time{}
	if tp.readTimeout != serial.NoTimeout {
		deadline = time.Now().Add(tp.readTimeout)
	}
	if err := tp.conn.SetReadDeadline(deadline); err != nil {
		return 0, err
	}
	return tp.conn.Read(p)
}

func (tp *TcpPort) Write(p []byte) (n int, err error) {
	return tp.conn.Write(p)
}

func (tp *TcpPort) Drain() error {
	return errors.New("not supported")
}

func (tp *TcpPort) ResetInputBuffer() error {
	return errors.New("not supported")
}

func (tp *TcpPort) ResetOutputBuffer() error {
	return errors.New("not supported")
}

func (tp *TcpPort) SetDTR(dtr bool) error {
	return errors.New("not supported")
}

func (tp *TcpPort) SetRTS(rts bool) error {
	return errors.New("not supported")
}

func (tp *TcpPort) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return nil, errors.New("not supported")
}

func (tp *TcpPort) SetReadTimeout(t time.Duration) error {
	tp.readTimeout = t
	return nil
}

func (tp *TcpPort) Close() error {
	return tp.conn.Close()
}

func (tp *TcpPort) Break(time.Duration) error {
	return errors.New("not supported")
}
