package read

import (
	"github.com/stretchr/testify/require"
	"server/pkg/logger"
	"testing"
)

func TestLoadTables(t *testing.T) {
	logger.NewZapLog("./tool.log", logger.Config{
		Level:   -1,
		Console: true,
	})
	err := LoadTables("../target")
	require.NoError(t, err)
}
