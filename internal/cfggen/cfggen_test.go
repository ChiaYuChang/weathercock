package cfggen_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/cfggen"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestReadConf(t *testing.T) {
	file, err := os.Open("../../.env")
	require.NoError(t, err)
	require.NotNil(t, file)

	// Create a source viper instance and read .env into it
	src := viper.New()
	src.SetConfigType("env")
	err = src.ReadConfig(file)
	require.NoError(t, err)

	w := cfggen.NewCfgGen(src) // Pass the sourceViper to NewCfgGen
	w.AddLLMConfig()
	w.AddLoggerConfig()
	w.AddKeywordExtractorConfig()

	buf := bytes.NewBuffer(nil)
	err = w.WriteTo(buf, "json")
	require.NoError(t, err)

	data, err := io.ReadAll(buf)
	require.NoError(t, err)
	t.Log(string(data))
}
