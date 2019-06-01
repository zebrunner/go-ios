package diagnostics

import "github.com/danielpaulus/go-ios/usbmux"

const serviceName = "com.apple.mobile.diagnostics_relay"

type Connection struct {
	muxConn    *usbmux.MuxConnection
	plistCodec *usbmux.PlistCodec
}

func New(deviceID int, udid string) *Connection {
	port := usbmux.StartService(deviceID, udid, serviceName)
	var screenShotrConn Connection
	screenShotrConn.muxConn = usbmux.NewUsbMuxConnection()
	responseChannel := make(chan []byte)

	plistCodec := usbmux.NewPlistCodecFromMuxConnection(screenShotrConn.muxConn, responseChannel)
	screenShotrConn.plistCodec = plistCodec
	screenShotrConn.muxConn.Connect(deviceID, port, plistCodec)

	return &screenShotrConn
}

func (diagnosticsConn *Connection) AllValues() allDiagnosticsResponse {
	allReq := diagnosticsRequest{"All"}
	diagnosticsConn.muxConn.Send(allReq)
	response := <-diagnosticsConn.plistCodec.ResponseChannel
	return diagnosticsfromBytes(response)
}

func (diagnosticsConn *Connection) Close() {
	closeReq := diagnosticsRequest{"Goodbye"}
	diagnosticsConn.muxConn.Send(closeReq)
	<-diagnosticsConn.plistCodec.ResponseChannel
	diagnosticsConn.muxConn.Close()

}
