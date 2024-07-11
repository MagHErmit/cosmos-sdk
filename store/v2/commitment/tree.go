package commitment

import (
	"errors"
	"io"

	ics23 "github.com/cosmos/ics23/go"

	coresstore "cosmossdk.io/core/store"
	snapshotstypes "cosmossdk.io/store/v2/snapshots/types"
)

// ErrorExportDone is returned by Exporter.Next() when all items have been exported.
var ErrorExportDone = errors.New("export is complete")

// Tree is the interface that wraps the basic Tree methods.
type Tree interface {
	// Write writes a batch of key-value pairs to the tree.
	Write(pairs coresstore.KVPairs) error
	// GetLatestVersion returns the latest version of the tree.
	GetLatestVersion() uint64

	// Hash returns the hash of the latest saved version of the tree.
	Hash() []byte

	// WorkingHash returns the working hash of the tree.
	WorkingHash() []byte

	// LoadVersion loads the tree at the given version.
	LoadVersion(version uint64) error
	// Commit commits the current state to the tree.
	Commit() ([]byte, uint64, error)
	// SetInitialVersion sets the initial version of the tree.
	SetInitialVersion(version uint64) error
	// GetProof returns a proof for the given key and version.
	GetProof(version uint64, key []byte) (*ics23.CommitmentProof, error)

	// Get attempts to retrieve a value from the tree for a given version.
	//
	// NOTE: This method only exists to support migration from IAVL v0/v1 to v2.
	// Once migration is complete, this method should be removed and/or not used.
	Get(version uint64, key []byte) ([]byte, error)

	Prune(version uint64) error
	Export(version uint64) (Exporter, error)
	Import(version uint64) (Importer, error)

	io.Closer
}

// Exporter is the interface that wraps the basic Export methods.
type Exporter interface {
	Next() (*snapshotstypes.SnapshotIAVLItem, error)

	io.Closer
}

// Importer is the interface that wraps the basic Import methods.
type Importer interface {
	Add(*snapshotstypes.SnapshotIAVLItem) error
	Commit() error

	io.Closer
}
