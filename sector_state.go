package sealing

type SectorState string

const (
	UndefinedSectorState SectorState = ""

	// happy path
	Empty          SectorState = "Empty"
	WaitDeals      SectorState = "WaitDeals"     // waiting for more pieces (deals) to be added to the sector
	Packing        SectorState = "Packing"       // sector not in sealStore, and not on chain
	PreCommit1     SectorState = "PreCommit1"    // do PreCommit1
	DoPreCommit1   SectorState = "DoPreCommit1"
	PreCommit2     SectorState = "PreCommit2"    // do PreCommit1
	DoPreCommit2   SectorState = "DoPreCommit2"
	PreCommitting  SectorState = "PreCommitting" // on chain pre-commit
	PreCommitWait  SectorState = "PreCommitWait" // waiting for precommit to land on chain
	WaitSeed       SectorState = "WaitSeed"      // waiting for seed
	Commit1        SectorState = "Commit1"
	DoCommit1      SectorState = "DoCommit1"
	Commit2        SectorState = "Commit2"
	DoCommit2      SectorState = "DoCommit2"
	Committing     SectorState = "Committing"
	CommitWait     SectorState = "CommitWait" // waiting for message to land on chain
	FinalizeSector SectorState = "FinalizeSector"
	Proving        SectorState = "Proving"
	// error modes
	FailedUnrecoverable  SectorState = "FailedUnrecoverable"
	SealPreCommit1Failed SectorState = "SealPreCommit1Failed"
	SealPreCommit2Failed SectorState = "SealPreCommit2Failed"
	PreCommitFailed      SectorState = "PreCommitFailed"
	ComputeProofFailed   SectorState = "ComputeProofFailed"
	CommitFailed         SectorState = "CommitFailed"
	PackingFailed        SectorState = "PackingFailed"
	FinalizeFailed       SectorState = "FinalizeFailed"

	Faulty        SectorState = "Faulty"        // sector is corrupted or gone for some reason
	FaultReported SectorState = "FaultReported" // sector has been declared as a fault on chain
	FaultedFinal  SectorState = "FaultedFinal"  // fault declared on chain

	Removing     SectorState = "Removing"
	RemoveFailed SectorState = "RemoveFailed"
	Removed      SectorState = "Removed"
)
