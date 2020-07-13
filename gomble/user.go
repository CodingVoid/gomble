package gomble

import "github.com/CodingVoid/gomble/logger"
import "github.com/CodingVoid/gomble/mumbleproto"

type user struct {
	id uint32
	//name string
}

//var users = map[uint32]string{}

func GetUser(userid uint32) *user {
	//username := users[userid]
	//if username == "" {
	//	return nil
	//}
	return &user{
		id: userid,
		//name: username,
	}
}

//func CheckUserExists(userid uint32) bool {
//	if users[userid] != "" {
//		return true
//	}
//	return false
//}

// send Message to the user
func (u user) SendMessage(msg string) {
	pck := mumbleproto.TextMessage{
		Actor:     nil, //uint32
		Message:   &msg,
		TreeId:    nil, //[]uint32 {},
		Session:   []uint32{u.id},
		ChannelId: nil, //[]uint32{},
	}
	logger.Infof("Sending Message to user %d\n", u.id)
	if err := writeProto(&pck); err != nil {
		panic(err.Error())
	}
}
