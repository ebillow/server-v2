package db

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMSet(t *testing.T) {
	ctx := context.Background()
	v := make([]interface{}, 0)
	for i := 0; i < 10; i++ {
		v = append(v, fmt.Sprintf("role:%d", i), i)
	}
	err := Redis.MSet(ctx, v).Err()
	require.Error(t, err)
}

func TestPipeSet(t *testing.T) {
	ctx := context.Background()
	pipe := Redis.Pipeline()
	for i := 0; i < 10; i++ {
		pipe.Set(ctx, fmt.Sprintf("role:%d", i), i, time.Minute*30)
	}
	_, err := pipe.Exec(ctx)
	require.NoError(t, err)
}

func TestMGet(t *testing.T) {
	ctx := context.Background()
	v := make([]string, 0)
	for i := 0; i < 10; i++ {
		v = append(v, fmt.Sprintf("role:%d", i))
	}
	_, err := Redis.MGet(ctx, v...).Result()
	require.Error(t, err)
}

func TestPipeGet(t *testing.T) {
	ctx := context.Background()
	pipe := Redis.Pipeline()
	for i := 0; i < 10; i++ {
		pipe.Get(ctx, fmt.Sprintf("role:%d", i))
	}
	cmds, err := pipe.Exec(ctx)
	require.NoError(t, err)
	t.Log(cmds)
}

func TestSet(t *testing.T) {
	ctx := context.Background()
	err := Redis.SAdd(ctx, "s1", 1, 2, 3, 4, 5).Err()
	require.NoError(t, err)
	err = Redis.SAdd(ctx, "s2", 2, 3, 4, 5).Err()
	require.NoError(t, err)

	err = Redis.SAdd(ctx, "{s}1", 1, 2, 3, 4, 5).Err()
	require.NoError(t, err)
	err = Redis.SAdd(ctx, "{s}2", 2, 3, 4, 5).Err()
	require.NoError(t, err)

	rs, err := Redis.SUnion(ctx, "{s}1", "{s}2").Result()
	require.NoError(t, err)
	t.Log("union:", rs)
	rs, err = Redis.SUnion(ctx, "s1", "s2").Result()
	require.Error(t, err)

	rs, err = Redis.SInter(ctx, "{s}1", "{s}2").Result()
	require.NoError(t, err)
	t.Log("inter:", rs)
	rs, err = Redis.SInter(ctx, "s1", "s2").Result()
	require.Error(t, err)

	rs, err = Redis.SDiff(ctx, "{s}1", "{s}2").Result()
	require.NoError(t, err)
	t.Log("diff:", rs)
	rs, err = Redis.SDiff(ctx, "s1", "s2").Result()
	require.Error(t, err)

	rs2, err := Redis.SUnionStore(ctx, "{s}12", "{s}1", "{s}2").Result()
	require.NoError(t, err)
	t.Log("union store:", rs2)
	rs2, err = Redis.SUnionStore(ctx, "s12", "s1", "s2").Result()
	require.Error(t, err)
}

func TestList(t *testing.T) {
	ctx := context.Background()

	err := Redis.LPush(ctx, "l1", 1, 2, 3, 4, 5).Err()
	require.NoError(t, err)
	// err = Redis.LPush(ctx, "l2", 2, 3, 4, 5).Err()
	// require.NoError(t, err)

	err = Redis.LPush(ctx, "{l}1", 1, 2, 3, 4, 5).Err()
	require.NoError(t, err)
	// err = Redis.LPush(ctx, "{l}2", 2, 3, 4, 5).Err()
	// require.NoError(t, err)

	err = Redis.RPopLPush(ctx, "l1", "l2").Err()
	require.Error(t, err)
	err = Redis.RPopLPush(ctx, "{l}1", "{l}2").Err()
	require.NoError(t, err)

	err = Redis.BRPopLPush(ctx, "l1", "l2", time.Second).Err()
	require.Error(t, err)
	err = Redis.BRPopLPush(ctx, "{l}1", "{l}2", time.Second).Err()
	require.NoError(t, err)

	err = Redis.LMove(ctx, "l1", "l2", "RIGHT", "LEFT").Err()
	require.Error(t, err)
	err = Redis.LMove(ctx, "{l}1", "{l}2", "RIGHT", "LEFT").Err()
	require.NoError(t, err)

	err = Redis.BLMove(ctx, "l1", "l2", "RIGHT", "LEFT", time.Second).Err()
	require.Error(t, err)
	err = Redis.BLMove(ctx, "{l}1", "{l}2", "RIGHT", "LEFT", time.Second).Err()
	require.NoError(t, err)
}

func TestZSet(t *testing.T) {
	ctx := context.Background()

	err := Redis.ZAdd(ctx, "z1",
		redis.Z{Score: 10, Member: 1},
		redis.Z{Score: 20, Member: 2},
		redis.Z{Score: 30, Member: 2},
		redis.Z{Score: 40, Member: 3},
		redis.Z{Score: 50, Member: 4}).Err()
	require.NoError(t, err)

	err = Redis.ZAdd(ctx, "z2",
		redis.Z{Score: 200, Member: 2},
		redis.Z{Score: 300, Member: 2},
		redis.Z{Score: 400, Member: 3},
		redis.Z{Score: 500, Member: 4}).Err()
	require.NoError(t, err)

	err = Redis.ZAdd(ctx, "{z}1",
		redis.Z{Score: 10, Member: 1},
		redis.Z{Score: 20, Member: 2},
		redis.Z{Score: 30, Member: 2},
		redis.Z{Score: 40, Member: 3},
		redis.Z{Score: 50, Member: 4}).Err()
	require.NoError(t, err)

	err = Redis.ZAdd(ctx, "{z}2",
		redis.Z{Score: 200, Member: 2},
		redis.Z{Score: 300, Member: 2},
		redis.Z{Score: 400, Member: 3},
		redis.Z{Score: 500, Member: 4}).Err()
	require.NoError(t, err)

	err = Redis.ZUnion(ctx, redis.ZStore{
		Keys: []string{"z1", "z2"},
	}).Err()
	require.Error(t, err)

	rs, err := Redis.ZUnion(ctx, redis.ZStore{
		Keys: []string{"{z}1", "{z}2"},
	}).Result()
	require.NoError(t, err)
	t.Log("zunion:", rs)

	err = Redis.ZUnionWithScores(ctx, redis.ZStore{
		Keys: []string{"z1", "z2"},
	}).Err()
	require.Error(t, err)

	rs2, err := Redis.ZUnionWithScores(ctx, redis.ZStore{
		Keys: []string{"{z}1", "{z}2"},
	}).Result()
	require.NoError(t, err)
	t.Log("zunionwithscore:", rs2)

	err = Redis.ZUnionStore(ctx, "z12", &redis.ZStore{
		Keys: []string{"z1", "z2"},
	}).Err()
	require.Error(t, err)

	err = Redis.ZUnionStore(ctx, "{z}12", &redis.ZStore{
		Keys: []string{"{z}1", "{z}2"},
	}).Err()
	require.NoError(t, err)
}

func TestRName(t *testing.T) {
	ctx := context.Background()
	err := Redis.Set(ctx, "str1", 1111, time.Minute).Err()
	require.NoError(t, err)
	err = Redis.Rename(ctx, "str1", "str2").Err()
	require.Error(t, err)

	err = Redis.Set(ctx, "{str}1", 1111, time.Minute).Err()
	require.NoError(t, err)
	err = Redis.Rename(ctx, "{str}1", "{str}2").Err()
	require.NoError(t, err)

	err = Redis.Set(ctx, "str3", 1111, time.Minute).Err()
	require.NoError(t, err)
	err = Redis.RenameNX(ctx, "str3", "str4").Err()
	require.Error(t, err)

	err = Redis.Set(ctx, "{str}3", 1111, time.Minute).Err()
	require.NoError(t, err)
	err = Redis.RenameNX(ctx, "{str}3", "{str}4").Err()
	require.NoError(t, err)
}

func TestLua(t *testing.T) {
	var script = redis.NewScript(`
redis.call("set", KEYS[1], ARGV[1])
redis.call("set", KEYS[2], ARGV[1])
return "OK"
`)
	ctx := context.Background()
	_, err := script.Run(ctx, Redis, []string{"lua1", "lua2"}, 1111).Result()
	t.Log(err)
	require.Error(t, err)

	_, err = script.Run(ctx, Redis, []string{"{lua}1", "{lua}2"}, 1111).Result()
	// t.Log(err)
	require.NoError(t, err)
}

func TestSCan(t *testing.T) {
	ctx := context.Background()
	ret, sor, err := Redis.Scan(ctx, 0, "s*", 100).Result()
	require.NoError(t, err)
	t.Log(ret)
	for sor != 0 {
		ret, sor, err = Redis.Scan(ctx, 0, "s*", 100).Result()
		require.NoError(t, err)
		t.Log(ret)
	}
}

func TestFlushDB(t *testing.T) {
	//	ctx := context.Background()
	//	Redis.FlushDB(ctx)

	//	Redis.FlushAll(ctx)
}
