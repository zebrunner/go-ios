package forward

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/danielpaulus/go-ios/usbmux"
	log "github.com/sirupsen/logrus"
)

type iosproxy struct {
	tcpWriter io.Writer

	readBuffer []byte
}

func Forward(device usbmux.DeviceEntry, hostPort uint16, phonePort uint16) error {

	log.Infof("Start listening on port %d forwarding to port %d on device", hostPort, phonePort)
	l, err := net.Listen("tcp", "localhost:7777")

	go connectionAccept(l, device.DeviceID, phonePort)

	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	return nil
}

func connectionAccept(l net.Listener, deviceID int, phonePort uint16) {

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("Error accepting new connections.")
		}
		log.Info("New client connected")
		go startNewProxyConnection(conn, deviceID, phonePort)
	}
}

func startNewProxyConnection(clientConn net.Conn, deviceID int, phonePort uint16) {

	usbmuxConn := usbmux.NewUsbMuxConnection()
	//defer usbmuxConn.Close()
	var proxyConnection iosproxy

	buf := make([]byte, 4096)
	proxyConnection.readBuffer = buf

	proxyConnection.tcpWriter = clientConn
	muxError := usbmuxConn.Connect(deviceID, phonePort, &proxyConnection)
	if muxError != nil {
		log.Fatal(muxError)
	}
	log.Infof("Connected to port %d", phonePort)

	tcpbuf := make([]byte, 4096)

	go func() {
		for {
			n, err := clientConn.Read(tcpbuf)

			//print("read from tcp" + string(n))
			if err != nil {
				log.Error("Closing Client")
				clientConn.Close()
				return
			}
			usbmuxConn.Send(tcpbuf[:n])
		}
	}()
}

func (proxyConn *iosproxy) close() {

}

func (proxyConn *iosproxy) Encode(message interface{}) ([]byte, error) {
	return message.([]byte), nil
}
func (proxyConn *iosproxy) Decode(r io.Reader) error {
	tcpbuf := make([]byte, 4096)
	n, err := r.Read(tcpbuf)

	if err != nil {
		proxyConn.close()
		return err
	}

	_, writerErr := proxyConn.tcpWriter.Write(tcpbuf[:n])
	if writerErr != nil {
		log.Error("failed writing")
	}

	return nil
}
