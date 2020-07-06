package gomble

import (
	"crypto/tls"
	"net"

	"github.com/golang/protobuf/proto"

	"github.com/CodingVoid/gomble/logger"
	"github.com/CodingVoid/gomble/mumbleproto"
)

// Eventpuffer is a global channel in which multiple go routines write several events. The only one reading on this channel is eventRoutine()
var eventpuffer chan event

var Listener listener
type listener struct {
	OnPrivateMessageReceived func(e PrivateMessageReceivedEvent)
	OnChannelMessageReceived func(e ChannelMessageReceivedEvent)
	OnTrackEnded func(e TrackEndedEvent)
}

// conn is our tcp connection to the server. It is used by packagereader.go to read packages from mumble-server and by packagewriter to write packages to the mumble-server.
var conn *tls.Conn
var tlsconfig tls.Config

// audioConn is our udp connection to the server. It is used by audiopackagereader.go to read packages from mumble-server and by audiopackagewriter to write packages to the mumble-server
var audioConn *net.UDPConn

// Adress and port of mumble-server in syntax address:port
var addr string


// Initializes some settings for gomble and returns an Eventhandler which can be used to add event-listeners
// loglevel the loglevel to use e.g. logger.DEBUG, logger.INFO, logger.WARN, logger.ERROR, logger.FATAL
// addr the address of the mumble-server written like "192.168.178.150:64738"
func Init(loglevel logger.Loglevel, address string) {
	logger.SetLogLevel(loglevel)
	tlsconfig.InsecureSkipVerify = true
	addr = address
	audiocryptoconfig.tcpTunnelMode = true // our audio goes through the tcp tunnel, until we successfully got a UDP Ping answer from the mumble-server
}

// Initializes the Connection 
func Begin() {
	connl, err := tls.Dial("tcp", addr, &tlsconfig)
	if err != nil {
		logger.Fatalf("TLS Connection could not be established: " + err.Error() + "\n")
	}
	logger.Infof("TLS Connection established\n")
	conn = connl

	// Initialize mumble connection

	versionPacket := mumbleproto.Version {
		Version:   proto.Uint32(66304),
		Release:   proto.String("gomble"),
		Os:        proto.String("linux"),
		OsVersion: proto.String("amd64"),
	}
	authPacket := mumbleproto.Authenticate {
		Username: proto.String("gomble-bot"),
		Password: proto.String(""),
		Opus:     proto.Bool(true),
		Tokens:   nil,
	}

	logger.Debugf("Send Version\n")
	if err := writeProto(&versionPacket); err != nil {
		logger.Fatalf("Sending Version failed: " + err.Error() + "\n")
	}

	logger.Debugf("Send Authentification")
	if err := writeProto(&authPacket); err != nil {
		logger.Fatalf("Sending Authentification failed: " + err.Error() + "\n")
	}

	// mumble connection established
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logger.Fatalf("Error getting UDP Address: " + err.Error())
	}
	audioConn, err = net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		logger.Fatalf("Error DialUDP: " + err.Error())
	}

	// create the eventpuffer before anyone tries to write or read on it
	eventpuffer = make(chan event)

	logger.Debugf("Start pingRoutine\n")
	go pingRoutine()

	logger.Debugf("Start readRoutine\n")
	go readRoutine()

	eventRoutine()
}

// The eventRoutine reads on the eventpuffer channel and executes the corresponding callbacks specified by the library user
func eventRoutine() {
	// Go through each received event
	for e := range eventpuffer {
		switch e.(type) {
		case PrivateMessageReceivedEvent:
			logger.Debugf("Received Private Message Received event\n")
			if Listener.OnPrivateMessageReceived != nil {
				Listener.OnPrivateMessageReceived(e.(PrivateMessageReceivedEvent))
			}
		case ChannelMessageReceivedEvent:
			logger.Debugf("Received Channel Message Received event\n")
			if Listener.OnChannelMessageReceived != nil {
				Listener.OnChannelMessageReceived(e.(ChannelMessageReceivedEvent))
			}
		case TrackEndedEvent:
			logger.Debugf("Received TrackEndedEvent\n")
			if Listener.OnTrackEnded != nil {
				Listener.OnTrackEnded(e.(TrackEndedEvent))
			}
		default:
			logger.Errorf("Received unknown Event\n")
		}
	}
}
