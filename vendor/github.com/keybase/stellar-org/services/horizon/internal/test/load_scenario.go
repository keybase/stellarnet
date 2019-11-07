package test

import (
	"github.com/stellar/go/services/horizon/internal/test/scenarios"
)

func loadScenario(scenarioName string, includeHorizon bool) {
	stellarCorePath := scenarioName + "-core.sql"

	scenarios.Load(StellarCoreDatabaseURL(), stellarCorePath)

	if includeHorizon {
		horizonPath := scenarioName + "-horizon.sql"
		scenarios.Load(DatabaseURL(), horizonPath)
	}
}
