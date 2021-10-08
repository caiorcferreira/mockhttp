package mockhttp

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	t.Run("start mock server at specify port", func(t *testing.T) {
		ms := NewMockServer(WithPort(60000))
		ms.Start(t)

		addr := "localhost:60000"
		require.Eventually(t, func() bool {
			_, err := net.Dial("tcp", addr)
			return err == nil
		}, 2*time.Second, 200*time.Millisecond)
	})

	t.Run("start mock server at any port available", func(t *testing.T) {
		ms := NewMockServer()
		ms.Start(t)

		addr := fmt.Sprintf("localhost:%d", ms.Port())
		require.Eventually(t, func() bool {
			_, err := net.Dial("tcp", addr)
			return err == nil
		}, 2*time.Second, 200*time.Millisecond)
	})
}
