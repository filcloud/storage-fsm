package sealing

import "errors"

var (
	ErrNotAccepted       = errors.New("task not accepted")
	ErrWorkerBusy        = errors.New("worker is busy")
	ErrNoAvailableWorker = errors.New("no available worker")
	ErrNoWorkerHasSector = errors.New("no worker has requested sector")
)
