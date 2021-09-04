package gomble

import "github.com/CodingVoid/gomble/logger"
import "github.com/CodingVoid/gomble/mumbleproto"

type UserState struct {
    Name string
    Session uint32
    ChannelId uint32
}

var BotUserState UserState

func SwitchChannel(channelid uint32) {
    pck := mumbleproto.UserState{
        ChannelId: &channelid,
    }
    logger.Infof("Switching to Channel %d\n", channelid)
    if err := writeProto(&pck); err != nil {
        logger.Errorf("Error while switching channel: %s", err.Error())
    }
}

// send Message to the user
func SendMessageToUser(msg string, userid uint32) {
    pck := mumbleproto.TextMessage{
        Actor:     nil, //uint32
        Message:   &msg,
        TreeId:    nil, //[]uint32 {},
        Session:   []uint32{userid},
        ChannelId: nil, //[]uint32{},
    }
    logger.Infof("Sending Message to user with session-ID: %d\n", userid)
    if err := writeProto(&pck); err != nil {
        logger.Errorf("Error while sending Message to User: %s", err.Error())
    }
}

// send Message to the user
func SendMessageToChannel(msg string, channelid uint32) {
    pck := mumbleproto.TextMessage{
        Actor:     nil, // uint32
        Message:   &msg,
        TreeId:    nil, // []uint32 {},
        Session:   nil, // []uint32{},
        ChannelId: []uint32{channelid},
    }
    logger.Infof("Sending Message to Channel %d\n", channelid)
    if err := writeProto(&pck); err != nil {
        logger.Errorf("Error while sending Message to Channel: %s", err.Error())
    }
}
