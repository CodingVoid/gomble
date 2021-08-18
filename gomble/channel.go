package gomble

import "github.com/CodingVoid/gomble/logger"
import "github.com/CodingVoid/gomble/mumbleproto"

type Channel struct {
    id uint32
}

func GetChannel(channelid uint32) Channel {
    return Channel{
        id: channelid,
    }
}

// send Message to the user
func (c Channel) SendMessage(msg string) {
    pck := mumbleproto.TextMessage{
        Actor:     nil, // uint32
        Message:   &msg,
        TreeId:    nil, // []uint32 {},
        Session:   nil, // []uint32{},
        ChannelId: []uint32{c.id},
    }
    logger.Infof("Sending Message to Channel %d\n", c.id)
    if err := writeProto(&pck); err != nil {
        panic(err.Error())
    }
}
