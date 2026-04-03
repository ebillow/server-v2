package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitClickhouse(t *testing.T) {
	err := InitClickhouse([]string{"172.21.10.195:9100"}, "admin", "clickhouse1.0", "game_log_201")
	require.NoError(t, err)
}
