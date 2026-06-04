package role

import (
	"time"
)

type IComp interface {
	IsDirty() bool // 是否需要保存
	ClearDirty()   // 清除脏数据标志
}

// ICompTick 如果需要每秒update，就实现该接口
type ICompTick interface {
	OnTick(now time.Time, r *Role) // 每秒更新
}

// ICompMinute 如果需要每分update，就实现该接口
type ICompMinute interface {
	OnMinute(now time.Time, r *Role) // 更新
}

// ICompDataReset 如果需要每天定时重置数据，就实现该接口
type ICompDataReset interface {
	OnDataReset(r *Role) //
}

// ICompNewMonth 跨月数据重置
type ICompNewMonth interface {
	OnNewMonth(r *Role) //
}

// ICompNewDay 如果需要跨天时处理数据，就实现该接口
type ICompNewDay interface {
	OnNewDay(r *Role) //
}

// ICompOnline 上线的处理接口
type ICompOnline interface {
	Online(r *Role)
}

// ICompOffline 下线的处理接口
type ICompOffline interface {
	Offline(r *Role)
}

// ICompOnPay 如果需要处理支付，就实现该接口,通过payID判断该次支付是否与本模块相关，
type ICompOnPay interface {
	CanPay(index uint32, payID uint32, r *Role) bool                // 判断能否支付，，
	OnPay(index uint32, payID uint32, payFee float64, r *Role) bool // 支付成功回调给组件,已处理return true
}

// ICompAnyPay 任意支付都触发事件
type ICompAnyPay interface {
	AnyPay(payFee float64, r *Role)
}
