package nucliozap

import (
	"testing"

	"github.com/nuclio/nuclio-sdk"
)

func TestSimpleLogging(t *testing.T) {
	var baseLogger nuclio.Logger

	baseLogger, err := NewNuclioZap("test", DebugLevel)
	if err != nil {
		t.Error(err)
	}

	baseLogger.Debug("Hello there")
	baseLogger.Info("Hello there info")
	baseLogger.Warn("Hello there %s", "warning")
	baseLogger.DebugWith("Hello there", "a", 3, "b", "foo")
	baseLogger.Error("An error %s %d", "aaa", 30)

	childLogger1 := baseLogger.GetChild("child1").(nuclio.Logger)
	childLogger1.DebugWith("What", "a", 1)

	childLogger2 := childLogger1.GetChild("child2").(nuclio.Logger)
	childLogger2.DebugWith("Foo", "a", 1)
	childLogger2.DebugWith("Foo", "a", 1)
}
