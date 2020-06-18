package processors

import logpkg "github.com/stellar/go/support/log"

var log = logpkg.DefaultLogger.WithField("service", "expingest")

const maxBatchSize = 100000
