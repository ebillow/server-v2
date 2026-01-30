package lock

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"server/modules/cfg"
	"server/modules/db"
	"server/modules/util"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	err := db.InitRedis(cfg.Redis{
		Address: "127.0.0.1:6379",
		DB:      0,
	}, 0, false)
	if err != nil {
		panic(err)
	}

	InitPool(db.Redis)
	m.Run()
}

func TestOpLock(t *testing.T) {
	go func() {
		{
			time.Sleep(time.Second * 2)
			err := Do(db.Redis, "s1", func(ctx context.Context, tx *redis.Tx) error {
				// v := tx.Get(ctx, "s1").Val()
				_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
					t.Log("change s1")
					return pipe.Set(ctx, "s1", 3, 0).Err()
				})
				return err
			})
			require.NoError(t, err)
		}
	}()
	go func() {
		{
			err := Do(db.Redis, "s1", func(ctx context.Context, tx *redis.Tx) error {
				v := tx.Get(ctx, "s1").Val()
				t.Log("get s1", v)
				time.Sleep(time.Second * 4)
				_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
					i := util.ParseInt32(v)
					return pipe.Set(ctx, "s1", i+1, 0).Err()
				})
				return err
			})

			require.NoError(t, err)
		}
	}()
	time.Sleep(time.Second * 10)
}

func TestLockWithSavePipe(t *testing.T) {
	go func() {
		{
			time.Sleep(time.Second * 2)
			err := DoWithSavePipe(db.Redis, "s1", func(ctx context.Context, tx *redis.Tx, save redis.Pipeliner) error {
				// v := tx.Get(ctx, "s1").Val()
				t.Log("change s1")
				return save.Set(ctx, "s1", 3, 0).Err()
			})
			require.NoError(t, err)
		}
	}()
	go func() {
		{
			err := DoWithSavePipe(db.Redis, "s1", func(ctx context.Context, tx *redis.Tx, save redis.Pipeliner) error {
				v := tx.Get(ctx, "s1").Val()
				t.Log("get s1", v)
				time.Sleep(time.Second * 4)

				i := util.ParseInt32(v)
				return save.Set(ctx, "s1", i+1, 0).Err()
			})
			require.NoError(t, err)
		}
	}()
	time.Sleep(time.Second * 10)
}

func TestLockWrite2(t *testing.T) {
	err := Do(db.Redis, "s1", func(ctx context.Context, tx *redis.Tx) error {
		v := tx.Get(ctx, "s1").Val()
		t.Log("get s1", v)

		err := func() error {
			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				i := util.ParseInt32(v)
				return pipe.Set(ctx, "s1", i+1, 0).Err()
			})
			return err
		}()
		if err != nil {
			return err
		}
		err = func() error {
			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				i := util.ParseInt32(v)
				return pipe.Set(ctx, "s1", i+1, 0).Err()
			})
			return err
		}()
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
}

func TestLockHash(t *testing.T) {
	go func() {
		{
			time.Sleep(time.Second * 2)
			err := Do(db.Redis, "h1", func(ctx context.Context, tx *redis.Tx) error {
				// v := tx.Get(ctx, "s1").Val()
				_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
					t.Log("change h1")
					return pipe.HSet(ctx, "h1", "f1", 3).Err()
				})
				return err
			})
			require.NoError(t, err)
		}
	}()
	go func() {
		{
			err := Do(db.Redis, "h1", func(ctx context.Context, tx *redis.Tx) error {
				v := tx.HGet(ctx, "h1", "f1").Val()
				t.Log("get s1", v)
				time.Sleep(time.Second * 4)
				_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
					i := util.ParseInt32(v) + 1
					return pipe.HSet(ctx, "h1", "f1", i).Err()
				})
				return err
			})

			require.NoError(t, err)
		}
	}()
	time.Sleep(time.Second * 10)
}

// BenchmarkDo-8   	    6055	    183543 ns/op
func BenchmarkDo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Do(db.Redis, "s1", func(ctx context.Context, tx *redis.Tx) error {
			v, _ := tx.Get(ctx, "s1").Int()
			_, err := tx.Pipelined(ctx, func(pipe redis.Pipeliner) error {
				return pipe.Set(ctx, "s1", v+1, 0).Err()
			})
			return err
		})
	}
}

// BenchmarkScript-8   	   21866	     52855 ns/op
func BenchmarkScript(b *testing.B) {
	var sp = redis.NewScript(`
	redis.call("get", KEYS[1])
	redis.call("set", KEYS[1], ARGV[1], "EX", ARGV[2])
`)
	for i := 0; i < b.N; i++ {
		_, _ = sp.Run(context.Background(), db.Redis, []string{"s2"}, 1, 10).Result()
		// if err != nil {
		//	b.Log(err)
		// }
	}
}
