package sealing

import (
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

type mutator interface {
	apply(state *SectorInfo)
}

// globalMutator is an event which can apply in every state
type globalMutator interface {
	// applyGlobal applies the event to the state. If if returns true,
	//  event processing should be interrupted
	applyGlobal(state *SectorInfo) bool
}

// Global events

type SectorRestart struct{}

func (evt SectorRestart) applyGlobal(*SectorInfo) bool { return false }

type SectorFatalError struct{ error }

func (evt SectorFatalError) FormatError(xerrors.Printer) (next error) { return evt.error }

func (evt SectorFatalError) applyGlobal(state *SectorInfo) bool {
	log.Errorf("Fatal error on sector %d: %+v", state.SectorNumber, evt.error)
	// TODO: Do we want to mark the state as unrecoverable?
	//  I feel like this should be a softer error, where the user would
	//  be able to send a retry event of some kind
	return true
}

type SectorForceState struct {
	State SectorState
}

func (evt SectorForceState) applyGlobal(state *SectorInfo) bool {
	state.State = evt.State
	return true
}

// Normal path

type SectorStart struct {
	ID         abi.SectorNumber
	SectorType abi.RegisteredProof
	Pieces     []Piece
}

func (evt SectorStart) apply(state *SectorInfo) {
	state.SectorNumber = evt.ID
	state.Pieces = evt.Pieces
	state.SectorType = evt.SectorType
}

type SectorPacked struct{ FillerPieces []abi.PieceInfo }

func (evt SectorPacked) apply(state *SectorInfo) {
	for idx := range evt.FillerPieces {
		state.Pieces = append(state.Pieces, Piece{
			Piece:    evt.FillerPieces[idx],
			DealInfo: nil, // filler pieces don't have deals associated with them
		})
	}
}

type SectorPackingFailed struct{ error }

func (evt SectorPackingFailed) apply(*SectorInfo) {}

type SectorPreCommit1 struct {
	TicketValue   abi.SealRandomness
	TicketEpoch   abi.ChainEpoch
}

func (evt SectorPreCommit1) apply(state *SectorInfo) {
	state.TicketEpoch = evt.TicketEpoch
	state.TicketValue = evt.TicketValue
}

type SectorFinishPreCommit1 struct {
	PreCommit1Out storage.PreCommit1Out
}

func (evt SectorFinishPreCommit1) apply(state *SectorInfo) {
	state.PreCommit1Out = evt.PreCommit1Out
}

type SectorPreCommit2 struct {
}

func (evt SectorPreCommit2) apply(state *SectorInfo) {
}

type SectorFinishPreCommit2 struct {
	Sealed   cid.Cid
	Unsealed cid.Cid
}

func (evt SectorFinishPreCommit2) apply(state *SectorInfo) {
	commd := evt.Unsealed
	state.CommD = &commd
	commr := evt.Sealed
	state.CommR = &commr
}

type SectorSealPreCommitFailed struct{ Err error }

func (evt SectorSealPreCommitFailed) Error() string { return evt.Err.Error() }
func (evt SectorSealPreCommitFailed) FormatError(xerrors.Printer) (next error) { return evt.Err }
func (evt SectorSealPreCommitFailed) apply(si *SectorInfo) {
	si.InvalidProofs = 0 // reset counter
}

type SectorChainPreCommitFailed struct{ error }

func (evt SectorChainPreCommitFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorChainPreCommitFailed) apply(*SectorInfo)                        {}

type SectorPreCommitted struct {
	Message cid.Cid
}

func (evt SectorPreCommitted) apply(state *SectorInfo) {
	state.PreCommitMessage = &evt.Message
}

type SectorSeedReady struct {
	SeedValue abi.InteractiveSealRandomness
	SeedEpoch abi.ChainEpoch
}

func (evt SectorSeedReady) apply(state *SectorInfo) {
	state.SeedEpoch = evt.SeedEpoch
	state.SeedValue = evt.SeedValue
}

type SectorComputeProofFailed struct{ Err error }

func (evt SectorComputeProofFailed) Error() string { return evt.Err.Error() }
func (evt SectorComputeProofFailed) FormatError(xerrors.Printer) (next error) { return evt.Err }
func (evt SectorComputeProofFailed) apply(*SectorInfo)                        {}

type SectorCommitFailed struct{ error }

func (evt SectorCommitFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorCommitFailed) apply(*SectorInfo)                        {}

type SectorCommit1 struct {
}

func (evt SectorCommit1) apply(state *SectorInfo) {
}

type SectorFinishCommit1 struct {
	Commit1Out []byte
}

func (evt SectorFinishCommit1) apply(state *SectorInfo) {
	state.Commit1Out = evt.Commit1Out
}

type SectorCommit2 struct {
}

func (evt SectorCommit2) apply(state *SectorInfo) {
}

type SectorFinishCommit2 struct {
	Proof   []byte
}

func (evt SectorFinishCommit2) apply(state *SectorInfo) {
	state.Proof = evt.Proof
}

type SectorCommitted struct {
	Message cid.Cid
}

func (evt SectorCommitted) apply(state *SectorInfo) {
	state.CommitMessage = &evt.Message
}

type SectorProving struct{}

func (evt SectorProving) apply(*SectorInfo) {}

type SectorFinalized struct{}

func (evt SectorFinalized) apply(*SectorInfo) {}

type SectorFinalizeFailed struct{ error }

func (evt SectorFinalizeFailed) FormatError(xerrors.Printer) (next error) { return evt.error }
func (evt SectorFinalizeFailed) apply(*SectorInfo)                        {}

// Failed state recovery

type SectorRetrySeal struct{}

func (evt SectorRetrySeal) apply(state *SectorInfo) {}

type SectorRetryPreCommit struct{}

func (evt SectorRetryPreCommit) apply(state *SectorInfo) {}

type SectorRetryWaitSeed struct{}

func (evt SectorRetryWaitSeed) apply(state *SectorInfo) {}

type SectorRetryComputeProof struct{}

func (evt SectorRetryComputeProof) apply(state *SectorInfo) {}

type SectorRetryInvalidProof struct{}

func (evt SectorRetryInvalidProof) apply(state *SectorInfo) {
	state.InvalidProofs++
}

// Faults

type SectorFaulty struct{}

func (evt SectorFaulty) apply(state *SectorInfo) {}

type SectorFaultReported struct{ reportMsg cid.Cid }

func (evt SectorFaultReported) apply(state *SectorInfo) {
	state.FaultReportMsg = &evt.reportMsg
}

type SectorFaultedFinal struct{}
