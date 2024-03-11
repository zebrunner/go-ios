package ios

import (
	"fmt"
	"net"
	"time"

	"github.com/danielpaulus/go-ios/ios/http"

	"github.com/danielpaulus/go-ios/ios/xpc"
)

type connectMessage struct {
	BundleID            string
	ClientVersionString string
	MessageType         string
	ProgName            string
	LibUSBMuxVersion    uint32 `plist:"kLibUSBMuxVersion"`
	DeviceID            uint32
	PortNumber          uint16
}

func newConnectMessage(deviceID int, portNumber uint16) connectMessage {
	data := connectMessage{
		BundleID:            "go.ios.control",
		ClientVersionString: "go-usbmux-0.0.1",
		MessageType:         "Connect",
		ProgName:            "go-usbmux",
		LibUSBMuxVersion:    3,
		DeviceID:            uint32(deviceID),
		PortNumber:          portNumber,
	}
	return data
}

// Connect issues a Connect Message to UsbMuxd for the given deviceID on the given port
// enabling the newCodec for it.
// It returns an error containing the UsbMux error code should the connect fail.
func (muxConn *UsbMuxConnection) Connect(deviceID int, port uint16) error {
	msg := newConnectMessage(deviceID, Ntohs(port))
	muxConn.Send(msg)
	resp, err := muxConn.ReadMessage()
	if err != nil {
		return err
	}
	response := MuxResponsefromBytes(resp.Payload)
	if response.IsSuccessFull() {
		return nil
	}
	return fmt.Errorf("Failed connecting to service, error code:%d", response.Number)
}

// serviceConfigurations stores info about which DTX based services only execute a SSL Handshake
// and then go back to sending unencrypted data right after the handshake.
var serviceConfigurations = map[string]bool{
	"com.apple.instruments.remoteserver":                 true,
	"com.apple.accessibility.axAuditDaemon.remoteserver": true,
	"com.apple.testmanagerd.lockdown":                    true,
	"com.apple.debugserver":                              true,
}

// ConnectLockdown connects this Usbmux connection to the LockDown service that
// always runs on the device on the same port. The connect call needs the deviceID which can be
// retrieved from a DeviceList using the ListDevices function. After this function
// is done, the UsbMuxConnection cannot be used anymore because the same underlying
// network connection is used for talking to Lockdown. Sending usbmux commands would break it.
// It returns a new LockDownConnection.
func (muxConn *UsbMuxConnection) ConnectLockdown(deviceID int) (*LockDownConnection, error) {
	msg := newConnectMessage(deviceID, Lockdownport)
	err := muxConn.Send(msg)
	if err != nil {
		return &LockDownConnection{}, err
	}
	resp, err := muxConn.ReadMessage()
	if err != nil {
		return &LockDownConnection{}, err
	}
	response := MuxResponsefromBytes(resp.Payload)
	if response.IsSuccessFull() {
		return &LockDownConnection{muxConn.deviceConn, "", NewPlistCodec()}, nil
	}

	return nil, fmt.Errorf("Failed connecting to Lockdown with error code:%d", response.Number)
}

func ConnectToService(device DeviceEntry, serviceName string) (DeviceConnectionInterface, error) {
	startServiceResponse, err := StartService(device, serviceName)
	if err != nil {
		return nil, err
	}
	pairRecord, err := ReadPairRecord(device.Properties.SerialNumber)
	if err != nil {
		return nil, err
	}

	muxConn, err := NewUsbMuxConnectionSimple()
	if err != nil {
		return nil, fmt.Errorf("Could not connect to usbmuxd socket, is it running? %w", err)
	}
	err = muxConn.connectWithStartServiceResponse(device.DeviceID, startServiceResponse, pairRecord)
	if err != nil {
		return nil, err
	}
	return muxConn.ReleaseDeviceConnection(), nil
}

// ConnectToServiceTunnelIface connects to a service on an iOS17+ device using a XPC over HTTP2 connection
// It returns a new xpc.Connection
func ConnectToXpcServiceTunnelIface(device DeviceEntry, serviceName string) (*xpc.Connection, error) {
	port := device.Rsd.GetPort(serviceName)

	h, err := ConnectToHttp2(device, port)
	if err != nil {
		return nil, fmt.Errorf("ConnectToXpcServiceTunnelIface: failed to connect to http2: %w", err)
	}
	return CreateXpcConnection(h)
}

func ConnectToServiceTunnelIface(device DeviceEntry, serviceName string) (DeviceConnectionInterface, error) {
	port := device.Rsd.GetPort(serviceName)

	conn, err := connectToTunnel(device, port)
	if err != nil {
		return nil, fmt.Errorf("ConnectToServiceTunnelIface: failed to connect to tunnel: %w", err)
	}

	return NewDeviceConnectionWithConn(conn), nil
}

func ConnectToHttp2(device DeviceEntry, port int) (*http.HttpConnection, error) {
	addr, err := net.ResolveTCPAddr("tcp6", fmt.Sprintf("[%s]:%d", device.Address, port))
	if err != nil {
		return nil, fmt.Errorf("ConnectToHttp2: failed to resolve address: %w", err)
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("ConnectToHttp2: failed to dial: %w", err)
	}

	err = conn.SetKeepAlive(true)
	if err != nil {
		return nil, fmt.Errorf("ConnectToHttp2: failed to set keepalive: %w", err)
	}
	err = conn.SetKeepAlivePeriod(1 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("ConnectToHttp2: failed to set keepalive period: %w", err)
	}
	return http.NewHttpConnection(conn)
}

func connectToTunnel(device DeviceEntry, port int) (*net.TCPConn, error) {
	addr, err := net.ResolveTCPAddr("tcp6", fmt.Sprintf("[%s]:%d", device.Address, port))
	if err != nil {
		return nil, fmt.Errorf("connectToTunnel: failed to resolve address: %w", err)
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("connectToTunnel: failed to dial: %w", err)
	}

	err = conn.SetKeepAlive(true)
	if err != nil {
		return nil, fmt.Errorf("connectToTunnel: failed to set keepalive: %w", err)
	}
	err = conn.SetKeepAlivePeriod(1 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("connectToTunnel: failed to set keepalive period: %w", err)
	}

	return conn, nil
}

func ConnectToHttp2WithAddr(a string, port int) (*http.HttpConnection, error) {
	addr, err := net.ResolveTCPAddr("tcp6", fmt.Sprintf("[%s]:%d", a, port))
	if err != nil {
		return nil, fmt.Errorf("ConnectToHttp2WithAddr: failed to resolve address: %w", err)
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("ConnectToHttp2WithAddr: failed to dial: %w", err)
	}

	err = conn.SetKeepAlive(true)
	if err != nil {
		return nil, fmt.Errorf("ConnectToHttp2WithAddr: failed to set keepalive: %w", err)
	}
	err = conn.SetKeepAlivePeriod(1 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("ConnectToHttp2WithAddr: failed to set keepalive period: %w", err)
	}
	return http.NewHttpConnection(conn)
}

func CreateXpcConnection(h *http.HttpConnection) (*xpc.Connection, error) {
	err := initializeXpcConnection(h)
	if err != nil {
		return nil, fmt.Errorf("CreateXpcConnection: failed to initialize xpc connection: %w", err)
	}

	clientServerChannel := http.NewStreamReadWriter(h, http.ClientServer)
	serverClientChannel := http.NewStreamReadWriter(h, http.ServerClient)

	xpcConn, err := xpc.New(clientServerChannel, serverClientChannel, h)
	if err != nil {
		return nil, fmt.Errorf("CreateXpcConnection: failed to create xpc connection: %w", err)
	}

	return xpcConn, nil
}

// connectWithStartServiceResponse issues a Connect Message to UsbMuxd for the given deviceID on the given port
// enabling the newCodec for it. It also enables SSL on the new service connection if requested by StartServiceResponse.
// It returns an error containing the UsbMux error code should the connect fail.
func (muxConn *UsbMuxConnection) connectWithStartServiceResponse(deviceID int, startServiceResponse StartServiceResponse, pairRecord PairRecord) error {
	err := muxConn.Connect(deviceID, startServiceResponse.Port)
	if err != nil {
		return err
	}

	var sslerr error
	if startServiceResponse.EnableServiceSSL {
		if _, ok := serviceConfigurations[startServiceResponse.Service]; ok {
			sslerr = muxConn.deviceConn.EnableSessionSslHandshakeOnly(pairRecord)
		} else {
			sslerr = muxConn.deviceConn.EnableSessionSsl(pairRecord)
		}
		if sslerr != nil {
			return sslerr
		}
	}

	return nil
}

func ConnectLockdownWithSession(device DeviceEntry) (*LockDownConnection, error) {
	muxConnection, err := NewUsbMuxConnectionSimple()
	if err != nil {
		return nil, fmt.Errorf("USBMuxConnection failed with: %v", err)
	}
	defer muxConnection.ReleaseDeviceConnection()

	pairRecord, err := muxConnection.ReadPair(device.Properties.SerialNumber)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve PairRecord with error: %v", err)
	}

	lockdownConnection, err := muxConnection.ConnectLockdown(device.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("Lockdown connection failed with: %v", err)
	}
	resp, err := lockdownConnection.StartSession(pairRecord)
	if err != nil {
		return nil, fmt.Errorf("StartSession failed: %+v error: %v", resp, err)
	}
	return lockdownConnection, nil
}

func initializeXpcConnection(h *http.HttpConnection) error {
	csWriter := http.NewStreamReadWriter(h, http.ClientServer)
	ssWriter := http.NewStreamReadWriter(h, http.ServerClient)

	err := xpc.EncodeMessage(csWriter, xpc.Message{
		Flags: xpc.AlwaysSetFlag,
		Body:  map[string]interface{}{},
		Id:    0,
	})
	if err != nil {
		return fmt.Errorf("initializeXpcConnection: failed to encode message: %w", err)
	}

	_, err = xpc.DecodeMessage(csWriter) // TODO : figure out if need to act on this frame
	if err != nil {
		return fmt.Errorf("initializeXpcConnection: failed to decode message: %w", err)
	}

	err = xpc.EncodeMessage(ssWriter, xpc.Message{
		Flags: xpc.InitHandshakeFlag | xpc.AlwaysSetFlag,
		Body:  nil,
		Id:    0,
	})
	if err != nil {
		return fmt.Errorf("initializeXpcConnection: failed to encode message 2: %w", err)
	}

	_, err = xpc.DecodeMessage(ssWriter) // TODO : figure out if need to act on this frame
	if err != nil {
		return fmt.Errorf("initializeXpcConnection: failed to decode message 2: %w", err)
	}

	err = xpc.EncodeMessage(csWriter, xpc.Message{
		Flags: 0x201, // alwaysSetFlag | 0x200
		Body:  nil,
		Id:    0,
	})
	if err != nil {
		return fmt.Errorf("initializeXpcConnection: failed to encode message 3: %w", err)
	}

	_, err = xpc.DecodeMessage(csWriter) // TODO : figure out if need to act on this frame
	if err != nil {
		return fmt.Errorf("initializeXpcConnection: failed to decode message 3: %w", err)
	}

	return nil
}
