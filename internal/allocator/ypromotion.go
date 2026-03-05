package allocator

import (
	"github.com/rawbytedev/memory3d/internal/types"
)

type YPromotionStrategy interface {
	FindPromotion(size uint32, region *types.Region, maxPlanes uint8) *PromotionPlan
	ApplyPromotion(plan *PromotionPlan) (*types.Allocation, error)
	CalculateCost(plan *PromotionPlan) uint64
}

type PromotionPlan struct {
	RegionID  uint64
	Planes    []uint32 // Plane IDs to use
	Fragments []PromotionFragment
	TotalSize uint32
	Cost      uint64  // Gas cost
	Score     float64 // Quality score (higher is better)
}

type PromotionFragment struct {
	PlaneID uint32
	StartZ  uint16
	Size    uint16
}

type BestFitPromoter struct {
	allowNonAdjacent bool
	maxSearchDepth   int
}

func (b *BestFitPromoter) FindPromotion(size uint32, region *types.Region, maxPlanes uint8) *PromotionPlan {
	region.RLock()
	defer region.RUnlock()

	// Collect planes with free space
	freePlanes := make([]*types.Plane, 0, len(region.Planes))
	for _, plane := range region.Planes {
		if plane.FreeBytes() > 0 {
			freePlanes = append(freePlanes, plane)
		}
	}

	// Try to find best fit
	bestPlan := b.findBestFit(size, freePlanes, maxPlanes)
	if bestPlan != nil {
		bestPlan.RegionID = region.ID
		bestPlan.Cost = b.CalculateCost(bestPlan)
		bestPlan.Score = b.calculateScore(bestPlan)
	}

	return bestPlan
}

func (b *BestFitPromoter) findBestFit(size uint32, planes []*types.Plane, maxPlanes uint8) *PromotionPlan {
	// Try to find single plane with enough space
	for _, plane := range planes {
		if uint32(plane.FreeBytes()) >= size {
			startZ, ok := plane.Allocate(uint16(size))
			if ok {
				// Rollback the test allocation
				plane.Free(startZ)

				return &PromotionPlan{
					Planes: []uint32{plane.ID},
					Fragments: []PromotionFragment{{
						PlaneID: plane.ID,
						StartZ:  startZ,
						Size:    uint16(size),
					}},
					TotalSize: size,
				}
			}
		}
	}

	// Need multiple planes
	return b.findMultiPlaneFit(size, planes, maxPlanes)
}

func (b *BestFitPromoter) findMultiPlaneFit(size uint32, planes []*types.Plane, maxPlanes uint8) *PromotionPlan {
	// This is a bin-packing problem
	// Simplified: try adjacent planes first, then any planes

	// Sort planes by free space (descending)
	sortedPlanes := make([]*types.Plane, len(planes))
	copy(sortedPlanes, planes)
	// Note: Need to implement sorting by FreeBytes

	var fragments []PromotionFragment
	var usedPlanes []uint32
	remaining := size

	for _, plane := range sortedPlanes {
		if remaining == 0 {
			break
		}

		freeBytes := plane.FreeBytes()
		if freeBytes == 0 {
			continue
		}

		// Try to allocate as much as possible in this plane
		allocSize := uint16(min(uint32(freeBytes), remaining))
		startZ, ok := plane.Allocate(allocSize)
		if !ok {
			continue
		}

		// Rollback test allocation
		plane.Free(startZ)

		fragments = append(fragments, PromotionFragment{
			PlaneID: plane.ID,
			StartZ:  startZ,
			Size:    allocSize,
		})
		usedPlanes = append(usedPlanes, plane.ID)
		remaining -= uint32(allocSize)
	}

	if remaining > 0 {
		return nil
	}

	return &PromotionPlan{
		Planes:    usedPlanes,
		Fragments: fragments,
		TotalSize: size,
	}
}

func (b *BestFitPromoter) CalculateCost(plan *PromotionPlan) uint64 {
	// Base cost + penalty for each plane beyond first
	baseCost := uint64(types.GasBaseAllocation + types.GasPerByte*int(plan.TotalSize))
	promotionPenalty := uint64(len(plan.Planes)-1) * uint64(types.GasYPromotion)
	return baseCost + promotionPenalty
}

func (b *BestFitPromoter) calculateScore(plan *PromotionPlan) float64 {
	// Higher score is better
	score := 1.0

	// Penalize for many planes
	planePenalty := float64(len(plan.Planes)) * 0.1
	score -= planePenalty

	// Reward for adjacent planes
	if b.arePlanesAdjacent(plan.Planes) {
		score += 0.2
	}

	// Reward for large contiguous blocks within planes
	for _, frag := range plan.Fragments {
		if frag.Size > 1024 { // Large block
			score += 0.05
		}
	}

	return score
}

func (b *BestFitPromoter) arePlanesAdjacent(planes []uint32) bool {
	for i := 1; i < len(planes); i++ {
		if planes[i] != planes[i-1]+1 {
			return false
		}
	}
	return true
}
