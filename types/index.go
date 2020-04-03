package types

import (
	ld "github.com/piprate/json-gold/ld"
)

type signature struct {
	head   []*ld.Quad
	domain []string
}

func (s *signature) Head() []*ld.Quad {
	return s.head
}

func (s *signature) Domain() []string {
	return s.domain
}

// type fileIndex struct {
// 	mimes      []string
// 	signatures []query.Signature
// 	add        func(pathname []string, file *File)
// 	remove     func(pathname []string, file *File)
// }

// func (fi *fileIndex) Add(pathname []string, resource query.Resource) {
// 	switch file := resource.(type) {
// 	case *File:
// 		fi.add(pathname, file)
// 	}
// }

// func (fi *fileIndex) Remove(pathname []string, resource query.Resource) {
// 	switch file := resource.(type) {
// 	case *File:
// 		fi.remove(pathname, file)
// 	}
// }

// func (fi *fileIndex) Signatures() []query.Signature {
// 	return fi.signatures
// }

// func NewFileIndex(
// 	mimes []string,
// 	add, remove func(pathname []string, file *File),
// 	signatures []query.Signature,
// ) query.Index {
// 	return &fileIndex{
// 		mimes,
// 		signatures,
// 		add,
// 		remove,
// 	}
// }
