package gomble

//import "gomble/logger"

//const (
//	CELTAplha = iota
//	Ping = iota
//	Speex = iota
//	CELTBeta = iota
//	OPUS = iota
//)
//
//type UDPPackageType byte

//// UDP Audio read routine. It receives Audio packages and udp ping packets from mumble-server
//func audioReadRoutine() {
//	for {
//		err := audioReadPackage()
//		if err != nil {
//			logger.Fatal("audioReadPackage error: " + err.Error())
//		}
//	}
//}
//
//// return package data, package type, package target
//func audioReadPackage() (error) {
//	var typeTarget [1]byte 
//	_, err := audioConn.Read(typeTarget[:])
//	if err != nil {
//		logger.Fatal("Could not read UDP package 1: " + err.Error())
//	}
//	// first 3 bits are packet type
//	pckType := (typeTarget[0] & 0xE0) >> 5
//	// remaining 5 bits are packet target
//	pckTarget := (typeTarget[0] & 0x1F)
//
//	switch pckType {
//	case Ping:
//		logger.Debug("Receveid UDP Ping packet")
//		timestamp, err := decodeVarint(audioConn)
//		if err != nil {
//			return err
//		}
//		break
//	case OPUS:
//		logger.Debug("Received UDP OPUS Audio Packet")
//		break
//	case CELTAplha:
//	case CELTBeta:
//	case Speex:
//		logger.Warn("Received Unsupported UDP Audio Package (CELT or Speex)")
//		break
//	default:
//		logger.Fatal("Received unknown UDP Package")
//	}
//	return nil
//}
