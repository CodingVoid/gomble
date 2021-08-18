package gomble

import (
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    "strconv"

    "github.com/golang/protobuf/proto"

    "github.com/CodingVoid/gomble/logger"
    "github.com/CodingVoid/gomble/mumbleproto"
)

func readRoutine() { // {{{
    for {
        var data [2048]byte
        pckType, pcksize, err := receivePacket(data[:])
        //logger.Debugf("Received entire Package")
        if err != nil {
            conn.Close()
            logger.Fatalf("Could not receive tcp packet: " + err.Error())
        }
        printReceivedPackage(pckType, data[:pcksize])
        handlePacket(pckType, data[:pcksize])
    }

} // }}}

func receivePacket(buffer []byte) (uint16, uint32, error) { // {{{
    var header [6]byte
    buffersize := len(buffer)

    if _, err := io.ReadFull(conn, header[:]); err != nil {
        return 0, 0, errors.New("receivePacket header could not be received\n")
    }
    pckType := binary.BigEndian.Uint16(header[:2])
    pckLen := binary.BigEndian.Uint32(header[2:])

    //logger.Debugf("Read Package Header: pckType: %d pckLen: %d\n", pckType, pckLen)

    if pckLen > uint32(buffersize) {
        return 0, 0, errors.New("receivePacket buffer was to small, buffersize: " + strconv.Itoa(buffersize) + " pckLen: " + strconv.Itoa(int(pckLen)))
    }

    if _, err := io.ReadFull(conn, buffer[:pckLen]); err != nil {
        return 0, 0, errors.New("receivePacket data could not be received\n")
    }

    return pckType, pckLen, nil
} // }}}

func handlePacket(pckType uint16, data []byte) { // {{{
    switch pckType {
        // Version
    case 0:
        var pck mumbleproto.Version
        proto.Unmarshal(data, &pck)
        // murmur only supports ocb2, but grumble also supports golangs secretbox package which uses XSalsa20 and Poly1305 to encrypt and authenticate messages
        for _, mode := range pck.GetCryptoModes() {
            // we use "XSalsa20-Poly1305" encryption if supported by Server (probably only grumble), otherwise use ocb2
            if mode == "XSalsa20-Poly1305" {
                audiocryptoconfig.cryptoMode = "XSalsa20-Poly1305"
            }
        }
        break
        // Voice Packet, ignore
    case 1:
        break
        // Crypt Setup
    case 15:
        var pck mumbleproto.CryptSetup
        proto.Unmarshal(data, &pck)
        // set config for sending audio data over udp
        if (audiocryptoconfig.cryptoMode == "XSalsa20-Poly1305") {
            audiocryptoconfig.cryptState.SetKey("XSalsa20-Poly1305", pck.GetKey(), pck.GetClientNonce(), pck.GetServerNonce())
        } else {
            audiocryptoconfig.cryptState.SetKey("OCB2-AES128", pck.GetKey(), pck.GetClientNonce(), pck.GetServerNonce())
        }
        break
        // Channel State
    case 7:
        var pck mumbleproto.ChannelState
        proto.Unmarshal(data, &pck)
        break
        // User State
    case 9:
        var pck mumbleproto.UserState
        proto.Unmarshal(data, &pck)
        break
        // Server sync
    case 5:
        var pck mumbleproto.ServerSync
        proto.Unmarshal(data, &pck)
        break
        // CodecVersion
    case 21:
        var pck mumbleproto.CodecVersion
        proto.Unmarshal(data, &pck)
        // PermissionQuery
    case 20:
        var pck mumbleproto.PermissionQuery
        proto.Unmarshal(data, &pck)
    case 24:
        // ServerConfig
        var pck mumbleproto.ServerConfig
        proto.Unmarshal(data, &pck)
        // Ping
    case 3:
        var pck mumbleproto.Ping
        proto.Unmarshal(data, &pck)
        // TextMessage
    case 11:
        var pck mumbleproto.TextMessage
        proto.Unmarshal(data, &pck)
        if pck.GetChannelId() == nil {
            mre := PrivateMessageReceivedEvent{
                Actor:   pck.GetActor(),
                Message: pck.GetMessage(),
            }
            eventpuffer <- mre
        } else {
            mre := ChannelMessageReceivedEvent{
                Actor:   pck.GetActor(),
                Message: pck.GetMessage(),
                Channel: pck.GetChannelId()[0],
            }
            eventpuffer <- mre
        }
        /*
        if (strings.Contains(pck.GetMessage(), "test")) {
            sendMessage("send back")
        }
        */
        // UserRemove
    case 8:
        var pck mumbleproto.UserRemove
        proto.Unmarshal(data, &pck)
    default:
        logger.Fatalf("unknown msg type: %d\n", pckType)
        break
    }
} // }}}

// for heavy debugging
func printReceivedPackage(pckType uint16, data []byte) { // {{{
    var out string
    switch pckType {
        // Version
    case 0:
        out += "Received packageType: Version (0)\n"
        var pck mumbleproto.Version
        proto.Unmarshal(data, &pck)
        out += "CryptoModes:"
        for _, mode := range pck.GetCryptoModes() {
            out += " " + mode
        }
        out += "\n"
        out += fmt.Sprintf("Version: %d\n", pck.GetVersion())
        out += fmt.Sprintf("OS: %s\n", pck.GetOs())
        out += fmt.Sprintf("Release: %s\n", pck.GetRelease())
        out += fmt.Sprintf("OSVersion: %s\n", pck.GetOsVersion())
        break
        // Voice Packet, ignore
    case 1:
        //out += "Received packageType: Voice (1)\n"
        break
        // Crypt Setup
    case 15:
        out += "Received packageType: CryptSetup (15)\n"
        var pck mumbleproto.CryptSetup
        proto.Unmarshal(data, &pck)
        out += formatByteArray("Key: ", pck.GetKey())
        out += formatByteArray("ClientNonce: ", pck.GetClientNonce())
        out += formatByteArray("ServerNonce: ", pck.GetServerNonce())
        break
        // Channel State
    case 7:
        out += "Received packageType: ChannelState (7)\n"
        var pck mumbleproto.ChannelState
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("Temporary?: %d\n", pck.GetTemporary())
        out += fmt.Sprintf("Can-Enter?: %d\n", pck.GetCanEnter())
        out += fmt.Sprintf("IsEnterRestricted?: %d\n", pck.GetIsEnterRestricted())
        out += fmt.Sprintf("Name: %s\n", pck.GetName())
        out += fmt.Sprintf("Description: %s\n", pck.GetDescription())
        out += fmt.Sprintf("Parent: %d\n", pck.GetParent())
        out += fmt.Sprintf("Max-Users: %d\n", pck.GetMaxUsers())
        out += fmt.Sprintf("Channel-ID: %d\n", pck.GetChannelId())
        out += fmt.Sprintf("Position: %d\n", pck.GetPosition())
        out += formatByteArray("GetDescriptionHash: ", pck.GetDescriptionHash())
        out += formatUint32Array("Links: ", pck.GetLinks())
        out += formatUint32Array("LinksAdd: ", pck.GetLinksAdd())
        out += formatUint32Array("LinksRemove: ", pck.GetLinksRemove())
        break
        // User State
    case 9:
        out += "Received packageType: UserState (9)\n"
        var pck mumbleproto.UserState
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("Deaf?: %d\n", pck.GetDeaf())
        out += fmt.Sprintf("Mute?: %d\n", pck.GetMute())
        out += fmt.Sprintf("Suppress?: %d\n", pck.GetSuppress())
        out += fmt.Sprintf("Recording?: %d\n", pck.GetRecording())
        out += fmt.Sprintf("SelfDeaf?: %d\n", pck.GetSelfDeaf())
        out += fmt.Sprintf("SelfMute?: %d\n", pck.GetSelfMute())
        out += fmt.Sprintf("PrioritySpeaker?: %d\n", pck.GetPrioritySpeaker())
        out += fmt.Sprintf("Hash: %s\n", pck.GetHash())
        out += fmt.Sprintf("Name: %s\n", pck.GetName())
        out += fmt.Sprintf("Comment: %s\n", pck.GetComment())
        out += fmt.Sprintf("PluginIdentity: %s\n", pck.GetPluginIdentity())
        out += fmt.Sprintf("Actor: %d\n", pck.GetActor())
        out += fmt.Sprintf("Session: %d\n", pck.GetSession())
        out += fmt.Sprintf("UserId: %d\n", pck.GetUserId())
        out += fmt.Sprintf("ChannelId: %d\n", pck.GetChannelId())
        out += formatByteArray("Texture: ", pck.GetTexture())
        out += formatByteArray("TextureHash: ", pck.GetTextureHash())
        out += formatByteArray("CommentHash: ", pck.GetCommentHash())
        out += formatByteArray("PluginContext: ", pck.GetPluginContext())
        out += formatStringArray("TemporaryAccessTokens: ", pck.GetTemporaryAccessTokens())
        break
        // Server sync
    case 5:
        out += "Received packageType: ServerSync (5)\n"
        var pck mumbleproto.ServerSync
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("MaxBandwidth: %u\n", pck.GetMaxBandwidth())
        out += fmt.Sprintf("Session: %d\n", pck.GetSession())
        out += fmt.Sprintf("Permissions: %d\n", pck.GetPermissions())
        out += fmt.Sprintf("Welcome-Text: %s\n", pck.GetWelcomeText())
        break
        // CodecVersion
    case 21:
        out += "Received packageType: CodecVersion (21)\n"
        var pck mumbleproto.CodecVersion
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("Beta: %d\n", pck.GetBeta())
        out += fmt.Sprintf("Alpha: %d\n", pck.GetAlpha())
        out += fmt.Sprintf("Opus?: %d\n", pck.GetOpus())
        out += fmt.Sprintf("PreferAlpha: %d\n", pck.GetPreferAlpha())
        // PermissionQuery
    case 20:
        out += "Received packageType: PermissionQuery (20)\n"
        var pck mumbleproto.PermissionQuery
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("Permissions: %d\n", pck.GetPermissions())
        out += fmt.Sprintf("ChannelID: %d\n", pck.GetChannelId())
        out += fmt.Sprintf("Flush?: %d\n", pck.GetFlush())
    case 24:
        // ServerConfig
        out += "Received packageType: ServerConfig (24)\n"
        var pck mumbleproto.ServerConfig
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("MaxBandwidth: %d\n", pck.GetMaxBandwidth())
        out += fmt.Sprintf("MessageLength: %d\n", pck.GetMessageLength())
        out += fmt.Sprintf("ImageMessageLength: %d\n", pck.GetImageMessageLength())
        out += fmt.Sprintf("MaxUsers: %d\n", pck.GetMaxUsers())
        out += fmt.Sprintf("Welcome-Text: %s\n", pck.GetWelcomeText())
        out += fmt.Sprintf("Allow-HTML?: %d\n", pck.GetAllowHtml())
        // Ping
    case 3:
        out += "Received packageType: Ping (3)\n"
        var pck mumbleproto.Ping
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("Timestamp: %llu\n", pck.GetTimestamp())
        out += fmt.Sprintf("TcpPackets: %d\n", pck.GetTcpPackets())
        out += fmt.Sprintf("Good: %d\n", pck.GetGood())
        out += fmt.Sprintf("Late: %d\n", pck.GetLate())
        out += fmt.Sprintf("Lost: %d\n", pck.GetLost())
        out += fmt.Sprintf("Resync: %d\n", pck.GetResync())
        out += fmt.Sprintf("UdpPackets: %d\n", pck.GetUdpPackets())
        out += fmt.Sprintf("TcpPingAvg: %f\n", pck.GetTcpPingAvg())
        out += fmt.Sprintf("TcpPingVar: %f\n", pck.GetTcpPingVar())
        out += fmt.Sprintf("UdpPingAvg: %f\n", pck.GetUdpPingAvg())
        out += fmt.Sprintf("UdpPingVar: %f\n", pck.GetUdpPingVar())
        // TextMessage
    case 11:
        out += "Received packageType: TextMessage (11)\n"
        var pck mumbleproto.TextMessage
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("Actor: %u\n", pck.GetActor())
        out += fmt.Sprintf("Message: %s\n", pck.GetMessage())
        out += formatUint32Array("TreeId: ", pck.GetTreeId())
        out += formatUint32Array("Session: ", pck.GetSession())
        out += formatUint32Array("ChannelId: ", pck.GetChannelId())
        // UserRemove
    case 8:
        out += "Received packageType: UserRemove (8)\n"
        var pck mumbleproto.UserRemove
        proto.Unmarshal(data, &pck)
        out += fmt.Sprintf("Actor: %u\n", pck.GetActor())
        out += fmt.Sprintf("Session: %u\n", pck.GetSession())
        out += fmt.Sprintf("Ban?: %d\n", pck.GetBan())
        out += fmt.Sprintf("Reason: %s\n", pck.GetReason())
    default:
        logger.Fatalf("unknown msg type: %d\n", pckType)
        break
    }
    // don't print on voice packages. My eyes want to live...
    if pckType != 1 {
        logger.Debugf(out + "\n")
    }
} // }}}
