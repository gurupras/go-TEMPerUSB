package temperusb

import (
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	require := require.New(t)
	log.SetLevel(log.DebugLevel)

	temper, err := New("test")
	require.Nil(err)
	require.NotNil(temper)

	temp, err := temper.GetTemperature()
	require.Nil(err)
	require.NotZero(temp)
}
