package util

import (
	"context"
	"sync/atomic"
	"time"

	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

var isMaster atomic.Bool

func key(serName string) string {
	return "/election/master/" + serName
}

// IsMaster  查询当前是否为主节点的接口
func IsMaster() bool {
	return isMaster.Load()
}

// StartElection 开始选举。这是一个阻塞方法
func StartElection(ctx context.Context, cli *clientv3.Client, serName string, val string, ttl int) {
	prefix := key(serName)

	for {
		// 1. 创建一个 Session（会话）。
		// Session 会自动向 Etcd 发送心跳以维持租约 (KeepAlive)。
		// 如果网络断开或进程崩溃，租约到期后，Etcd 会自动删除该节点，释放主节点身份。
		session, err := concurrency.NewSession(cli, concurrency.WithTTL(ttl))
		if err != nil {
			zap.L().Error("创建 Etcd Session 失败, 准备重试", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}

		// 2. 创建选举对象
		election := concurrency.NewElection(session, prefix)

		zap.L().Info("开始参与主节点竞选...", zap.String("val", val))

		// 3. 参与竞选 (Campaign)
		// 这是一个阻塞操作！只有当本进程成功当选为主节点时，该方法才会返回。
		if err := election.Campaign(ctx, val); err != nil {
			zap.L().Error("竞选失败或退出", zap.Error(err))
			session.Close()

			// 如果 context 被取消，说明程序要退出了
			if ctx.Err() != nil {
				return
			}
			time.Sleep(time.Second)
			continue
		}

		// ---------------- 竞选成功，成为 Master ----------------
		isMaster.Store(true)
		zap.L().Info("竞选成功，当前节点已成为 Master!", zap.String("val", val))

		// 4. 监听会话状态或上下文退出
		select {
		case <-session.Done():
			// Session.Done() 被触发意味着租约失效（可能是网络问题导致心跳失败）
			zap.L().Warn("Etcd Session 过期，失去 Master 身份")
		case <-ctx.Done():
			// 程序主动退出
			zap.L().Info("收到退出信号，准备卸任 Master")
		}

		// ---------------- 失去 Master 身份 ----------------
		isMaster.Store(false)

		// 5. 主动卸任 (Resign)
		// 如果是主动退出，调用 Resign 可以立刻让出主节点，让其他节点迅速接管，而不需要等待 TTL 过期
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := election.Resign(ctxTimeout); err != nil {
			zap.L().Error("卸任 Master 失败", zap.Error(err))
		}
		cancel()
		session.Close()

		// 如果是程序退出，直接 return 结束循环
		if ctx.Err() != nil {
			return
		}
	}
}
