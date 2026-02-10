package robot

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"math/big"
	"server/pkg/crypt/dh"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"server/pkg/util"
	"server/robot/clinet"
)

const TimeOutMs = 300

// RegisteMsgHandle 注册消息处理函数
func RegisteMsgHandle() {
	clinet.RegistryMsg(msgid.MsgIDS2C_S2CInit, func() proto.Message { return &pb.S2CInit{} }, onInit)
	clinet.RegistryMsg(msgid.MsgIDS2C_S2CLogin, func() proto.Message { return &pb.S2CLogin{} }, onLogin)
	clinet.RegistryMsg(msgid.MsgIDS2C_S2CHeartBeat, func() proto.Message { return &pb.S2CHeartBeat{} }, onHeartbeat)
}

func onInit(msgBase proto.Message, ses *clinet.Session) {
	msg := msgBase.(*pb.S2CInit)
	if msg.Crypto {
		var err error
		var s2cKey []byte
		var c2sKey []byte
		if s2cPublic, ok := big.NewInt(0).SetString(msg.S2CPublic, 0); ok {
			s2cKey = dh.GetKey(ses.S2cPrivate, s2cPublic).Bytes()
		}
		if c2sPublic, ok := big.NewInt(0).SetString(msg.C2SPublic, 0); ok {
			c2sKey = dh.GetKey(ses.C2sPrivate, c2sPublic).Bytes()
		}
		err = ses.Init(c2sKey, s2cKey)
		if err != nil {
			zap.L().Error("Init fail", zap.Error(err))
			return
		}
	} else {
		err := ses.Init(nil, nil)
		if err != nil {
			zap.L().Error("Init fail", zap.Error(err))
			return
		}
	}

	r := ses.U.(*Robot)
	switch r.state {
	case Init:
		r.Login()
	case ReConn:
		r.ReConn()
	}

}

func onLogin(msgBase proto.Message, ses *clinet.Session) {
	msg := msgBase.(*pb.S2CLogin)
	r := ses.U.(*Robot)

	if msg.Code != pb.LoginCode_LCSuccess {
		zap.S().Warnf("%s login err:%s", r.acc, pb.LoginCode_name[int32(msg.Code)])
		return
	}

	r.onLoginSuccess(msg)
}

func onHeartbeat(msgBase proto.Message, ses *clinet.Session) {
	msg := msgBase.(*pb.S2CHeartBeat)

	r := ses.U.(*Robot)
	if r == nil {
		return
	}

	// fmt.Printf("game time out :%ds", util.GetNowTimeM()-msg.CliTime)

	if util.GetNowTimeM()-msg.CliTime > TimeOutMs {
		TimeOut(r.id)
	}
	if r.Data != nil {
		Active(r.Data.ID)
	}
}
