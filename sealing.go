package sealing

import (
	"context"
	"io"

	"github.com/filecoin-project/specs-storage/storage"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-padreader"
	"github.com/filecoin-project/go-statemachine"
	"github.com/filecoin-project/sector-storage/ffiwrapper"
	"github.com/filecoin-project/sector-storage/storiface"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/crypto"
)

const SectorStorePrefix = "/sectors"

var log = logging.Logger("sectors")

type SealingAPI interface {
	StateWaitMsg(context.Context, cid.Cid) (MsgLookup, error)
	StateComputeDataCommitment(ctx context.Context, maddr address.Address, sectorType abi.RegisteredSealProof, deals []abi.DealID, tok TipSetToken) (cid.Cid, error)
	StateSectorPreCommitInfo(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok TipSetToken) (*miner.SectorPreCommitOnChainInfo, error)
	StateSectorGetInfo(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tok TipSetToken) (*miner.SectorOnChainInfo, error)
	StateMinerSectorSize(context.Context, address.Address, TipSetToken) (abi.SectorSize, error)
	StateMinerWorkerAddress(ctx context.Context, maddr address.Address, tok TipSetToken) (address.Address, error)
	StateMinerDeadlines(ctx context.Context, maddr address.Address, tok TipSetToken) (*miner.Deadlines, error)
	StateMinerInitialPledgeCollateral(context.Context, address.Address, abi.SectorNumber, TipSetToken) (big.Int, error)
	StateMarketStorageDeal(context.Context, abi.DealID, TipSetToken) (market.DealProposal, error)
	SendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, gasPrice big.Int, gasLimit int64, params []byte) (cid.Cid, error)
	ChainHead(ctx context.Context) (TipSetToken, abi.ChainEpoch, error)
	ChainGetRandomness(ctx context.Context, tok TipSetToken, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) (abi.Randomness, error)
	ChainReadObj(context.Context, cid.Cid) ([]byte, error)
}

type SectorManager interface {
	SetSealing(sealing *Sealing)

	SectorSize() abi.SectorSize

	ReadPiece(context.Context, io.Writer, abi.SectorID, storiface.UnpaddedByteIndex, abi.UnpaddedPieceSize, abi.SealRandomness, cid.Cid) error

	SealPreCommit1(ctx context.Context, sector abi.SectorID, ticket abi.SealRandomness, pieces []abi.PieceInfo) error
	SealPreCommit2(ctx context.Context, sector abi.SectorID, pc1o storage.PreCommit1Out) error
	SealCommit1(ctx context.Context, sector abi.SectorID, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) error
	SealCommit2(ctx context.Context, sector abi.SectorID, c1o storage.Commit1Out) error
	FinalizeSector(ctx context.Context, sector abi.SectorID) error

	storage.Storage
	storage.Prover

	CheckProvable(ctx context.Context, spt abi.RegisteredSealProof, sectors []abi.SectorID) ([]abi.SectorID, error)
}

type Sealing struct {
	api    SealingAPI
	events Events

	maddr address.Address

	sealer  SectorManager
	sectors *statemachine.StateGroup
	sc      SectorIDCounter
	verif   ffiwrapper.Verifier

	pcp PreCommitPolicy
}

func New(api SealingAPI, events Events, maddr address.Address, ds datastore.Batching, sealer SectorManager, sc SectorIDCounter, verif ffiwrapper.Verifier, pcp PreCommitPolicy) *Sealing {
	s := &Sealing{
		api:    api,
		events: events,

		maddr:  maddr,
		sealer: sealer,
		sc:     sc,
		verif:  verif,
		pcp:    pcp,
	}

	s.sectors = statemachine.New(namespace.Wrap(ds, datastore.NewKey(SectorStorePrefix)), s, SectorInfo{})

	return s
}

func (m *Sealing) Sectors() *statemachine.StateGroup {
	return m.sectors
}

func (m *Sealing) Run(ctx context.Context) error {
	if err := m.restartSectors(ctx); err != nil {
		log.Errorf("%+v", err)
		return xerrors.Errorf("failed load sector states: %w", err)
	}

	return nil
}

func (m *Sealing) Stop(ctx context.Context) error {
	return m.sectors.Stop(ctx)
}

func (m *Sealing) AllocatePiece(size abi.UnpaddedPieceSize) (sectorID abi.SectorNumber, offset uint64, err error) {
	if (padreader.PaddedSize(uint64(size))) != size {
		return 0, 0, xerrors.Errorf("cannot allocate unpadded piece")
	}

	sid, err := m.sc.Next()
	if err != nil {
		return 0, 0, xerrors.Errorf("getting sector number: %w", err)
	}

	err = m.sealer.NewSector(context.TODO(), m.MinerSector(sid)) // TODO: Put more than one thing in a sector
	if err != nil {
		return 0, 0, xerrors.Errorf("initializing sector: %w", err)
	}

	// offset hard-coded to 0 since we only put one thing in a sector for now
	return sid, 0, nil
}

func (m *Sealing) SealPiece(ctx context.Context, size abi.UnpaddedPieceSize, r io.Reader, sectorID abi.SectorNumber, d DealInfo) error {
	log.Infof("Seal piece for deal %d", d.DealID)

	ppi, err := m.sealer.AddPiece(ctx, m.MinerSector(sectorID), []abi.UnpaddedPieceSize{}, size, r)
	if err != nil {
		return xerrors.Errorf("adding piece to sector: %w", err)
	}

	rt, err := ffiwrapper.SealProofTypeFromSectorSize(m.sealer.SectorSize())
	if err != nil {
		return xerrors.Errorf("bad sector size: %w", err)
	}

	return m.NewSector(sectorID, rt, []Piece{
		{
			Piece:    ppi,
			DealInfo: &d,
		},
	})
}

// newSector accepts a slice of pieces which will have a deal associated with
// them (in the event of a storage deal) or no deal (in the event of sealing
// garbage data)
func (m *Sealing) NewSector(sid abi.SectorNumber, rt abi.RegisteredSealProof, pieces []Piece) error {
	log.Infof("Start sealing %d", sid)
	return m.sectors.Send(uint64(sid), SectorStart{
		ID:         sid,
		Pieces:     pieces,
		SectorType: rt,
	})
}

func (m *Sealing) MinerSector(num abi.SectorNumber) abi.SectorID {
	mid, err := address.IDFromAddress(m.maddr)
	if err != nil {
		panic(err)
	}

	return abi.SectorID{
		Number: num,
		Miner:  abi.ActorID(mid),
	}
}

func (m *Sealing) Address() address.Address {
	return m.maddr
}

func (m *Sealing) SectorIDCounter() SectorIDCounter {
	return m.sc
}
