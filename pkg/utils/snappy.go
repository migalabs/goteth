package utils

import (
	"time"

	"github.com/golang/snappy"
	"github.com/pkg/errors"
)

type SSZserializable interface {
	MarshalSSZ() ([]byte, error)
	SizeSSZ() int
}

// compilation of metrics about compression
type CompressionMetrics struct {
	SSZsize           uint32
	SnappySize        uint32
	CompressionTime   time.Duration
	DecompressionTime time.Duration
}

// Retrieve the compression metrics about any sszSerializable block
func CompressConsensusSignedBlock(sszB SSZserializable) (CompressionMetrics, error) {
	sszSize := uint32(sszB.SizeSSZ()) // cast to uint32 to keep the previous format
	sszBytes, err := sszB.MarshalSSZ()
	if err != nil {
		return CompressionMetrics{}, errors.Wrap(err, "unable to encode block to ssz")
	}
	compSize, compTime, decompTime, err := snappyCompress(sszBytes)
	if err != nil {
		return CompressionMetrics{}, errors.Wrap(err, "unable to compress block with snappy")
	}

	cMetrics := CompressionMetrics{
		SSZsize:           sszSize,
		SnappySize:        compSize,
		CompressionTime:   compTime,
		DecompressionTime: decompTime,
	}
	return cMetrics, nil
}

// main compression method<
func snappyCompress(rawB []byte) (compSize uint32, compTime, decompTime time.Duration, err error) {
	// compression
	startT := time.Now()
	compressBytes := snappy.Encode(nil, rawB)
	compTime = time.Since(startT)

	// get compressed size
	compSize = uint32(len(compressBytes))

	// decompression
	startT = time.Now()
	_, err = snappy.Decode(nil, rawB)
	decompTime = time.Since(startT)
	if err != nil {
		return compSize, compTime, decompTime, errors.Wrap(err, "unable to decode block")
	}
	return compSize, compTime, decompTime, nil
}
