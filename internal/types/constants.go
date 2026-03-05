package types

const (
	// Memory dimensions
	MaxXRegions = 256
	MaxYPlanes  = 65535 // 64K planes per region
	MaxZOffset  = 65535 // 64KB per plane
	PlaneSize   = 65535 // 64KB

	// Gas constants
	GasBaseAllocation = 3
	GasPerByte        = 1
	GasYPromotion     = 10
	GasRegionChange   = 100
)
