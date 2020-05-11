package main

import (
	"context"
	"errors"

	badger "github.com/dgraph-io/badger/v2"
	rpc "github.com/underlay/pkgs/rpc"
	types "github.com/underlay/pkgs/types"
)

// ErrParentNotPackage indicates that the parent of the given path either does not exist, or is not a package
var ErrParentNotPackage = errors.New("Invalid path: parent is not a package")

// r is a *Package, *Assertion, or *File --- NOT a *Reference
func (server *Server) set(ctx context.Context, key []string, r types.Resource, txn *badger.Txn) error {
	if len(key) > 0 {
		parentItem, err := txn.Get(getKey(key[:len(key)-1]))
		if err != nil {
			return err
		} else if parentItem.UserMeta() != byte(types.PackageType) {
			return ErrParentNotPackage
		}
	}

	old, err := getResource(key, txn)
	if err == badger.ErrKeyNotFound {
		err = nil
	} else if err != nil {
		return err
	}

	if old != nil && old.URI() == r.URI() {
		return nil
	}

	oldPackage, isOldPackage := old.(*types.Package)
	oldAssertion, isOldAssertion := old.(*types.Assertion)
	oldFile, isOldFile := old.(*types.File)
	switch r := r.(type) {
	case *types.Package:
		if isOldPackage {
			err = server.diffChildren(ctx, key, r, oldPackage, txn)
		} else {
			err = server.setChildren(ctx, key, r, txn)
		}
	case *types.Assertion:
		if isOldAssertion {
			r.Created = oldAssertion.Created
		} else if isOldPackage {
			err = deleteChildren(key, oldPackage, txn)
		}
	case *types.File:
		if isOldFile {
			r.Created = oldFile.Created
		} else if isOldPackage {
			err = deleteChildren(key, oldPackage, txn)
		}
	}
	if err != nil {
		return err
	}

	rpc.Set(key, r)
	err = setResource(key, r, txn)
	if err != nil {
		return err
	}

	return nil
}

func (server *Server) setChildren(
	ctx context.Context,
	key []string, pkg *types.Package,
	txn *badger.Txn,
) error {
	for _, p := range pkg.Members.Packages {
		child, err := server.parse(ctx, p)
		if err != nil {
			return err
		}
		err = server.setChildren(ctx, append(key, p.Title), child, txn)
		if err != nil {
			return err
		}

		rpc.Set(key, child)
		err = setResource(key, child, txn)
		if err != nil {
			return err
		}
	}

	for _, a := range pkg.Members.Assertions {
		childKey := append(key, a.Name())
		rpc.Set(childKey, a)
		err := setResource(childKey, a, txn)
		if err != nil {
			return err
		}
	}

	for _, f := range pkg.Members.Files {
		childKey := append(key, f.Name())
		rpc.Set(childKey, f)
		err := setResource(childKey, f, txn)
		if err != nil {
			return err
		}
	}

	return nil
}

func (server *Server) diffChildren(
	ctx context.Context,
	key []string,
	pkg, oldPkg *types.Package,
	txn *badger.Txn,
) error {
	packages, assertions, files := pkg.Members.Packages, pkg.Members.Assertions, pkg.Members.Files

	packageExists := make([]bool, len(packages))
	for _, old := range oldPkg.Members.Packages {
		i, new := pkg.SearchPackages(old.Title, false)
		if new == nil {
			childKey := append(key, old.Title)
			oldChild, err := getPackage(childKey, txn)
			if err != nil {
				return err
			}

			err = deleteChildren(childKey, oldChild, txn)
			if err != nil {
				return err
			}

			rpc.Delete(childKey, oldChild)
			err = txn.Delete(getKey(childKey))
			if err != nil {
				return err
			}
		} else {
			packageExists[i] = true
			if new.ID != old.ID {
				childKey := append(key, old.Title)
				newChild, err := server.parse(ctx, new)
				if err != nil {
					return err
				}
				oldChild, err := getPackage(childKey, txn)
				if err != nil {
					return err
				}
				err = server.diffChildren(ctx, childKey, newChild, oldChild, txn)
				if err != nil {
					return err
				}

				rpc.Set(childKey, newChild)
				err = setResource(childKey, newChild, txn)
				if err != nil {
					return err
				}
			}
		}
	}

	for i, p := range packages {
		if !packageExists[i] {
			childKey := append(key, p.Title)
			newChild, err := server.parse(ctx, p)
			if err != nil {
				return err
			}

			err = server.setChildren(ctx, childKey, newChild, txn)
			if err != nil {
				return err
			}

			rpc.Set(childKey, newChild)
			err = setResource(childKey, newChild, txn)
			if err != nil {
				return err
			}
		}
	}

	assertionExists := make([]bool, len(assertions))
	for _, old := range oldPkg.Members.Assertions {
		name := old.Name()
		i, new := pkg.SearchAssertions(name, old.Resource == "")
		if new == nil {
			childKey := append(key, name)
			rpc.Delete(childKey, old)
			err := txn.Delete(getKey(childKey))
			if err != nil {
				return err
			}
		} else {
			assertionExists[i] = *new == *old
		}
	}

	for i, a := range assertions {
		if !assertionExists[i] {
			childKey := append(key, a.Name())
			rpc.Set(childKey, a)
			err := setResource(childKey, a, txn)
			if err != nil {
				return err
			}
		}
	}

	fileExists := make([]bool, len(files))
	for _, old := range oldPkg.Members.Files {
		name := old.Name()
		i, new := pkg.SearchFiles(name, old.Resource == "")
		if new == nil {
			childKey := append(key, name)
			rpc.Delete(childKey, old)
			err := txn.Delete(getKey(childKey))
			if err != nil {
				return err
			}
		} else {
			fileExists[i] = *new == *old
		}
	}

	for i, f := range files {
		if !fileExists[i] {
			childKey := append(key, f.Name())
			rpc.Set(childKey, f)
			err := setResource(childKey, assertions[i], txn)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
