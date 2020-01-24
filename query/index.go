package query

// An Index is the interface for database indices
type Index interface {
	Open()
	Close() // might change to Commit()
	Add(pathname []string, resource Resource)
	Remove(pathname []string, resource Resource)
	Signatures() []Signature
}
