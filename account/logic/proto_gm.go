package logic

func registeGameMsgHandle() {
	// network.RegisterHandle(pb.MsgIDS2S_Gm2AccBindAcc, func() proto.Message { return &pb.MsgBindAcc{} }, onGameBindAcc)
	// network.RegisterHandle(pb.MsgIDS2S_S2SUnBindAcc, func() proto.Message { return &pb.MsgBindAcc{} }, onGameUnBindAcc)
	// network.RegisterHandle(msgid.MsgIDS2S_S2SRoleOffline, onS2SRoleOffline)
}

//
// func onGameBindAcc(msgBase proto.Message, s *snet.Session) {
// 	msg := msgBase.(*pb.MsgBindAcc)
// 	if msg == nil {
// 		return
// 	}
//
// 	login.ConnectAcc(msg, s)
// }
//
// func onGameUnBindAcc(msgBase proto.Message, s *snet.Session) {
// 	//msg := msgBase.(*pb.MsgBindAcc)
// 	//if msg == nil || msg.SdkToBind == pb.ESdkNumber_GuestOverSeas {
// 	//	return
// 	//}
// 	//
// 	//connectAcc, err := login.DisconnectAcc(msg.RoleGuid, msg.GuestAccID, share.GetRawAcc(msg.SdkToBind, 0, msg.AccToBind))
// 	//if err != nil {
// 	//	logger.Warnf("disconnect err:%v", err)
// 	//	return
// 	//}
// 	//msg.ConnectedAcc = connectAcc
// 	//s.SendPB(pb.MsgIDS2S_S2SUnBindAcc, msg)
// }
//
// func onS2SRoleOffline(msgBase proto.Message, s *snet.Session) {
// 	msg := msgBase.(*pb.S2SRoleOffline)
// 	if msg == nil {
// 		return
// 	}
//
// 	login.PostEvt(login.EvtParam{
// 		Op:       login.OpRoleClear,
// 		RoleInfo: msg,
// 		Sn:       msg.Sn,
// 	})
// }
