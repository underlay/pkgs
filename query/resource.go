package query

import cid "github.com/ipfs/go-cid"

// Resource is interface type for resources (packages, messages, and files)
type Resource interface {
	Type() ResourceType
	ETag() (cid.Cid, string)
	URI() string
}

// ResourceType is an enum for resource types
type ResourceType uint8

const (
	// PackageType the ResourceType for Packages
	PackageType ResourceType = iota
	// MessageType the ResourceType for Messages
	MessageType
	// FileType is the ResourceType for Files
	FileType
)
