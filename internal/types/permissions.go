package types

// Permission type
type Permission byte

const (
	PermissionRead Permission = 1 << iota
	PermissionWrite
	PermissionExecute
	PermissionFree
	PermissionShare
)

// CheckPermission checks if a permission is set
func (p Permissions) CheckPermission(perm Permission) bool {
	switch perm {
	case PermissionRead:
		return p.Readable
	case PermissionWrite:
		return p.Writable
	case PermissionExecute:
		return p.Executable
	case PermissionFree:
		return true // Always allowed to free own allocations
	case PermissionShare:
		return p.Shared
	default:
		return false
	}
}

// CanAccess checks if address can be accessed with given permission
func CanAccess(addr Address3D, perm Permission, region *Region) bool {
	if region == nil {
		return false
	}

	// Check region permissions
	if !region.Permissions.CheckPermission(perm) {
		return false
	}

	// Check if address is within region bounds
	plane := region.GetPlane(addr.Y)
	if plane == nil {
		return false
	}

	// Check if offset is within plane
	if addr.Z >= plane.Size {
		return false
	}

	// For write operations, check if location is allocated
	if perm == PermissionWrite {
		return plane.IsAllocated(addr.Z)
	}

	return true
}
