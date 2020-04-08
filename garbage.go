package sealing

import (
	"context"
	"io"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-actors/actors/abi"

	nr "github.com/filecoin-project/storage-fsm/lib/nullreader"
)

func (m *Sealing) pledgeReader(size abi.UnpaddedPieceSize) io.Reader {
	return io.LimitReader(&nr.Reader{}, int64(size))
}

func (m *Sealing) PledgeSectorWithNull(ctx context.Context, sectorID abi.SectorID, existingPieceSizes []abi.UnpaddedPieceSize, sizes ...abi.UnpaddedPieceSize) ([]abi.PieceInfo, error) {
	if len(sizes) == 0 {
		return nil, nil
	}

	log.Infof("Pledge %d, contains %+v", sectorID, existingPieceSizes)

	out := make([]abi.PieceInfo, len(sizes))
	for i, size := range sizes {
		ppi, err := m.sealer.AddPiece(ctx, sectorID, existingPieceSizes, size, m.pledgeReader(size))
		if err != nil {
			return nil, xerrors.Errorf("add piece: %w", err)
		}

		existingPieceSizes = append(existingPieceSizes, size)

		out[i] = ppi
	}

	return out, nil
}

func (m *Sealing) PledgeSectorWithExisting(ctx context.Context, sectorID abi.SectorID) ([]abi.PieceInfo, error) {
	log.Infof("Pledge %d using existing", sectorID)

	size := abi.PaddedPieceSize(m.sealer.SectorSize()).Unpadded()

	// Here size 0 means using existing unsealed sector
	ppi, err := m.sealer.AddPiece(ctx, sectorID, []abi.UnpaddedPieceSize{}, 0, m.pledgeReader(size))
	if err != nil {
		return nil, xerrors.Errorf("add piece using existing: %w", err)
	}
	return []abi.PieceInfo{
		{Size: ppi.Size, PieceCID: ppi.PieceCID},
	}, nil
}

func (m *Sealing) PledgeSector(useExisting ...bool) error {
	return xerrors.New("not implemented")
}
