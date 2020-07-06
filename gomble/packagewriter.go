package gomble

// if anything is written to the TCP-Connection with the mumble-server, it happens here. 

import (
	"time"
	"encoding/binary"
	"net"

	"github.com/golang/protobuf/proto"

	"github.com/CodingVoid/gomble/logger"
	"github.com/CodingVoid/gomble/mumbleproto"
)

func writeProto(msg proto.Message) error {// {{{
	protoType := mumbleproto.MessageType(msg)

	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return sendPacket(protoType, data)
}// }}}

func sendPacket(protoType uint16, data []byte) error {// {{{
	var header [6]byte
	binary.BigEndian.PutUint16(header[:], protoType)
	binary.BigEndian.PutUint32(header[2:], uint32(len(data)))
	if _, err := conn.Write(header[:]); err != nil {
		logger.Warnf("Error Writing Package header with type: %u\n", protoType)
		return err
	}
	if _, err := conn.Write(data); err != nil {
		logger.Warnf("Error Writing Package data with type: %u\n", protoType)
		return err
	}
	return nil
}// }}}

// pingRoutine sends ping packets (TCP and UDP) to the server at regular intervals. TCP because the Server needs to receive one every 30 seconds, otherwise we get kicked. And UDP to check if the UDP connection to the Server is still working. If not we send Audio Packages over TCP Tunnel.
func pingRoutine() {// {{{
	tcpPing := time.NewTicker(time.Second * 20)
	udpPing := time.NewTicker(time.Second * 4)
	defer tcpPing.Stop()
	defer udpPing.Stop()

	var timestamp uint64
	var tcpPingAvg float32
	var tcpPingVar float32
	var tcpPacketsReceived uint32
	tcpPingPacket := mumbleproto.Ping{
		Timestamp:  &timestamp,
		TcpPackets: &tcpPacketsReceived,
		TcpPingAvg: &tcpPingAvg,
		TcpPingVar: &tcpPingVar,
	}

	tUdp, tTcp := time.Now(), time.Now()
	for {

		select {
			//case <-end:
			//	return
		case tTcp = <-tcpPing.C:
			timestamp = uint64(tTcp.UnixNano())
			//tcpPingAvg = math.Float32frombits(atomic.LoadUint32(&tcpPingAvg))
			//tcpPingVar = math.Float32frombits(atomic.LoadUint32(&tcpPingVar))
			writeProto(&tcpPingPacket)
			break
		case tUdp = <-udpPing.C:
			var header byte = 0x20 // ping header for audiopacket
			data := encodeVarint(tUdp.UnixNano()) // write timestamp as unix timestamp in nanoseconds

			var all []byte
			all = append(all[:], header)
			all = append(all[:], data[:]...)

			//udp_encrypt (currently ocb2-aes)
			//var encryptall []byte = make([]byte, len(all)+audiocryptoconfig.cryptState.Overhead())
			var encryptall []byte = make([]byte, len(all)+4) // ocb overhead always 4 bytes
			audiocryptoconfig.cryptState.Encrypt(encryptall[:], all[:])

			// send UDP Ping
			n, err := audioConn.Write(encryptall[:])
			if err != nil {
				logger.Warnf("Could not send UDP Ping Message\n")
				audiocryptoconfig.tcpTunnelMode = true
				continue
			}
			if n < len(encryptall) {
				logger.Warnf("Could not send full encrypted buffer of Ping Message\n")
				audiocryptoconfig.tcpTunnelMode = true
				continue
			}

			// Receive UDP Ping Packet Answer
			audioConn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
			var crypted []byte = make([]byte, 1024)
			n, err = audioConn.Read(crypted[:]) // read entire udp package

			crypted = crypted[:n]

			if err != nil {
				if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
					audiocryptoconfig.tcpTunnelMode = true
					logger.Infof("UDP ping timeout reached. Change back to TCP Tunnel\n")
					audiocryptoconfig.tcpTunnelMode = true
					continue
				}
				logger.Fatalf("Could not read UDP package 1: " + err.Error())
			}

			// ocb_decrypt
			var plain []byte = make([]byte, len(crypted))
			audiocryptoconfig.cryptState.Decrypt(plain[:], crypted[:])
			// Now remove tag and other overhead stuff
			plain = plain[:len(plain)-4] // ocb overhead always 4 bytes

			// first 3 bits are packet type
			pckType := (plain[0] & 0xE0) >> 5
			// remaining 5 bits are packet target
			_ = (plain[0] & 0x1F)

			if pckType == Ping {
				timestamp, err := decodeVarint(plain[1:])
				if err != nil {
					logger.Errorf("decodeVarint udp packet error: " + err.Error() + "\n")
				}
				logger.Debugf("Received UDP Ping Packet, timestamp as number: %d, timestamp: %s\n", timestamp, time.Unix(0, timestamp).String())
				audiocryptoconfig.tcpTunnelMode = false
			}
		}
	}
}// }}}
