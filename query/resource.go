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
	// Package the ResourceType for Packages
	Package ResourceType = iota
	// Message the ResourceType for Messages
	Message
	// File is the ResourceType for Files
	File
)
