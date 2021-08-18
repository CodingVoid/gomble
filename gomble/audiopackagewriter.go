package gomble

import (
    "encoding/binary"
    "math"

    "github.com/CodingVoid/gomble/cryptstate"
    "github.com/CodingVoid/gomble/logger"
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
    forceTcpTunnelMode bool

    // UDP CryptoState (currently OCB2)
    cryptState cryptstate.CryptState
    cryptoMode string
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

    // opus encoded audio data
    var terminateBit int64
    if last {
        terminateBit = 1
    } else {
        terminateBit = 0
    }

    opusHeader := terminateBit<<13 | int64(len(opusPayload))
    opusHeaderVar := encodeVarint(opusHeader)

    // tcp stack header
    audioheadersize := 1 + len(audioSequenceNum) + len(opusHeaderVar)
    pcksize := audioheadersize + len(opusPayload)
    var header []byte = make([]byte, 6+audioheadersize)
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

    // Do we tunnel audio over TCP or do we send it over UDP
    if audiocryptoconfig.tcpTunnelMode {
        logger.Debugf("Send via TCP Tunnel\n")
        n, err := conn.Write(all)
        if err != nil {
            logger.Fatalf("Error writing to tls connection\n")
        }
        if n < len(all) {
            logger.Fatalf("Did not write as much as expected\n")
        }
    } else {
        // encrypt ocb2-aes or XSalsa20-Poly1305
        var allencrypted []byte = make([]byte, len(all)+audiocryptoconfig.cryptState.Overhead()-6) // UDP we don't need 2 bytes package type and 4 bytes package length (encryptionOverhead-2-4)
        audiocryptoconfig.cryptState.Encrypt(allencrypted[:], all[6:])
        logger.Debugf("Send via UDP\n")
        // send ocb2-aes or XSalsa20-Poly1305 encrypted udp package
        n, err := audioConn.Write(allencrypted[:])

        if err != nil {
            logger.Fatalf("Error writing to UDP connection")
        }
        if n < len(allencrypted) {
            logger.Fatalf("Did not write as much as expected")
        }
    }

}
