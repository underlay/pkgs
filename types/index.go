package types

// An Index is the interface for database indices
type Index interface {
	Add(pathname []string, resource Resource)
	Remove(pathname []string, resource Resource)
	Signatures() []Signature
}
