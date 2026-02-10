package logic

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"math/big"
	"server/gateway/session"
	"server/pkg/crypt/dh"
	"server/pkg/pb"
	"server/pkg/pb/msgid"
	"server/pkg/thread"
	"strconv"
)

func init() {
	session.C().Msg(msgid.MsgIDC2S_C2SInit, onNetInit) // 初始化

	session.S().Msg(msgid.MsgIDS2S_S2SResLogin, onLoginSuccess)
	session.S().Msg(msgid.MsgIDS2S_S2SS2GtDisconnect, onDisconnect)
}

func onNetInit(msgBase proto.Message, ses *session.Session) {
	IsCrypto := false
	if IsCrypto {
		msg, ok := msgBase.(*pb.C2SInit)
		if !ok {
			zap.S().Warnf("msg type err %s", thread.FuncCaller(1))
			return
		}

		S2CPrivate, S2CPublic := dh.Exchange()
		zap.S().Debugf("%s s2c public=%s, private=%s", ses.String(), S2CPublic.String(), S2CPrivate.String())
		k, err := strconv.Atoi(msg.S2CPublic)
		if err != nil {
			zap.S().Warnf("s2c key err:%v", err)
			return
		}
		s2cKey := dh.GetKey(S2CPrivate, big.NewInt(int64(k)))

		C2SPrivate, C2SPublic := dh.Exchange()
		zap.S().Debugf("%s c2s public=%s, private=%s", ses.String(), C2SPublic.String(), C2SPrivate.String())
		k, err = strconv.Atoi(msg.C2SPublic)
		if err != nil {
			zap.S().Warnf("c2s key err:%v", err)
			return
		}
		c2sKey := dh.GetKey(C2SPrivate, big.NewInt(int64(k)))

		err = ses.Init(c2sKey.Bytes(), s2cKey.Bytes())
		if err != nil {
			zap.S().Warnf("init err:%v", err)
			return
		}
		ses.Send(&pb.S2CInit{
			S2CPublic: S2CPublic.String(),
			C2SPublic: C2SPublic.String(),
			Crypto:    IsCrypto,
		})
		// zap.S().Infof("%s send init msg", ses.String())
	} else {
		_ = ses.Init([]byte(""), []byte(""))
		ses.Send(&pb.S2CInit{
			Crypto: false,
		})
		// zap.S().Infof("%s send init msg", ses.String())
	}
}

func onLoginSuccess(msgBase proto.Message, ses *session.Session) {
	msg := msgBase.(*pb.S2SResLogin)
	ses.GameID = msg.GameID

	ses.Send(msg.Res)
}

func onDisconnect(msgBase proto.Message, ses *session.Session) {
	msg := msgBase.(*pb.S2SS2GtDisconnect)
	ses.Close(msg.Why)
}
