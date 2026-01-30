package logic

import ()

//
// func initCity(r *Robot) {
// 	clinet.RegistryMsg(pb.MsgIDS2C_S2CCity, func() proto.Message { return &pb.CityProto{} }, onCityProto)
// 	r.taskMgr.Add(int64(util.RandRangeFloat(20, 30)), cityTask)
// }
//
// func onCityProto(msgBase proto.Message, ses *clinet.Session) {
// 	msg := msgBase.(*pb.CityProto)
// 	r := ses.U.(*Robot)
// 	switch msg.Op {
// 	case pb.CityProto_OpFix:
// 		bld := r.Data.City.CurCity.Buildings[msg.Build]
// 		if bld != nil {
// 			bld.DestroyLevel = msg.DestroyLevel
// 		}
// 	case pb.CityProto_OpPartLevelUp:
// 		r.Data.City.CurCity.Buildings[msg.Build] = msg.Bld
// 	case pb.CityProto_OpTaskChapterFinish:
// 		r.Data.City.CurCity.Buildings[msg.Build] = msg.Bld
// 	}
// }
//
// func cityTask(r *Robot) {
// 	if util.Rand(3000) {
// 		fixBuilding(r)
// 	} else if util.Rand(3000) {
// 		partLvUp(r)
// 	} else {
// 		buildingLvUp(r)
// 	}
// }
//
// func fixBuilding(r *Robot) {
// 	for bld, v := range r.Data.City.CurCity.Buildings {
// 		if v.DestroyLevel > 0 {
// 			r.Send(pb.MsgIDC2S_C2SCity, &pb.CityProto{Op: pb.CityProto_OpFix, Build: bld, DestroyLevel: v.DestroyLevel})
// 			return
// 		}
// 	}
// }
//
// func partLvUp(r *Robot) {
// 	for bld, v := range r.Data.City.CurCity.Buildings {
// 		if v.DestroyLevel == 0 {
// 			for i, p := range v.Parts {
// 				if p.Level < 10 && util.Rand(5000) {
// 					r.Send(pb.MsgIDC2S_C2SCity, &pb.CityProto{Op: pb.CityProto_OpPartLevelUp, Build: bld, Part: uint32(i)})
// 					return
// 				}
// 			}
// 		}
// 	}
// }
//
// func buildingLvUp(r *Robot) {
// 	for bld, v := range r.Data.City.CurCity.Buildings {
// 		if v.DestroyLevel == 0 {
// 			if util.Rand(5000) {
// 				r.Send(pb.MsgIDC2S_C2SCity, &pb.CityProto{Op: pb.CityProto_OpBuildingLevelUpStart, Build: bld})
// 				return
// 			}
// 		}
// 	}
// }
