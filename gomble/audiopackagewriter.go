package gomble

import (
	"math"
	"encoding/binary"

	"github.com/CodingVoid/gomble/logger"
	"github.com/CodingVoid/gomble/cryptstate"
)

// constants for audiotypes of mumble
const (
	CELTAplha = iota
	Ping
	Speex
	CELTBeta
	OPUS
)

type audioCryptoConfig struct {
	// ip:port
	server string

	tcpTunnelMode bool

	// UDP CryptoState (currently OCB2)
	cryptState cryptstate.CryptState
}

var audiocryptoconfig audioCryptoConfig

var sequenceNumber int64 = 0
func sendAudioPacket(opusPayload []byte, opusPayloadlen uint16, last bool) {
	var protoType uint16
	protoType = 1 // in case of TCP Tunnel

	// type is audio, header:
	var audioType, audioTarget byte
	audioType = OPUS
	audioTarget = 0
	audioHeader := (audioType << 5) | (audioTarget << 0)

	// Sequence Number varint encoden
	audioSequenceNum := encodeVarint(sequenceNumber)

	sequenceNumber = (sequenceNumber + 1) % math.MaxInt32 // sequence number increment
	//fmt.Printf("SequencenumberNum: %d ", sequenceNumber)
	//fmt.Printf("Sequencenumber: [% x]\n", audioSequenceNum)

	// opus encoded audio data
	var terminateBit int64
	if (last) {
		terminateBit = 1
	} else {
		terminateBit = 0
	}

	opusHeader := terminateBit << 13 | int64(len(opusPayload))
	opusHeaderVar := encodeVarint(opusHeader)

	// tcp stack header
	audioheadersize := 1 + len(audioSequenceNum) + len(opusHeaderVar)
	pcksize := audioheadersize + len(opusPayload)
	var header []byte = make([]byte, 6 + audioheadersize)
	binary.BigEndian.PutUint16(header[:], protoType)
	binary.BigEndian.PutUint32(header[2:6], uint32(pcksize)) // len(opusPayload) + len(audioHeader) + len(opusHeaderVar) + len(sequencenum)
	header[6] = audioHeader
	copy(header[7:], audioSequenceNum[:])
	copy(header[7+len(audioSequenceNum):], opusHeaderVar[:])

	var all []byte = make([]byte, len(header)+len(opusPayload))
	copy(all[:len(header)], header[:])
	copy(all[len(header):], opusPayload[:])
	logger.Debugf("header size: %d\n", len(header))
	logger.Debugf("payloa size: %d\n", len(opusPayload))
	logger.Debugf("entire size %d\n", len(all))
	//conn.Write(header[:])
	//conn.Write(opusPayload[:])

	// Do we tunnel audio over TCP or do we send it over UDP
	if audiocryptoconfig.tcpTunnelMode {
		logger.Debug("Send via TCP Tunnel\n")
		n, err := conn.Write(all)
		if err != nil {
			logger.Fatal("Error writing to tls connection\n")
		}
		if n < len(all) {
			logger.Fatal("Weniger geschrieben als gedacht\n")
		}
	} else {
		// encrypt ocb2-aes
		var allencrypted []byte = make([]byte, len(all)-2) //ocb2 overhead is always 4 byte, but for UDP we don't need 2 bytes package type and 4 bytes package length (4-2-4 = -2)
		audiocryptoconfig.cryptState.Encrypt(allencrypted[:], all[6:])
		logger.Debug("Send via UDP\n")
		// send ocb2-aes encrytped udp package
		n, err := audioConn.Write(allencrypted[:])

		if err != nil {
			logger.Fatal("Error writing to UDP connection")
		}
		if n < len(allencrypted) {
			logger.Fatal("Weniger geschrieben als gedacht")
		}
	}

}
