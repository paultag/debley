package main

import (
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"xi2.org/x/xz"

	"pault.ag/go/archive"
)

func ohshit(err error) {
	if err != nil {
		panic(err)
	}
}

type Node []string

func (n Node) String() string {
	return strings.Join(n, "/")
}

type Tripple struct {
	Root     Node
	Relation Node
	Target   Node
}

func (t Tripple) String() string {
	return fmt.Sprintf("<%s> <%s> <%s> .", t.Root, t.Relation, t.Target)
}

func PackageId(dist, suite, arch, name string) Node {
	return Node{dist, suite, arch, name}
}

func PackageTripples(dist, suite, arch string, pkg *archive.Package) ([]Tripple, error) {
	id := PackageId(dist, suite, arch, pkg.Package)

	var source string
	if pkg.Source != "" {
		source = strings.SplitN(pkg.Source, " ", 2)[0]
	} else {
		source = pkg.Package
	}

	ret := []Tripple{
		Tripple{Root: id, Relation: Node{"priority"}, Target: Node{"priority", pkg.Priority}},
		Tripple{Root: id, Relation: Node{"section"}, Target: Node{"section", pkg.Section}},
		Tripple{Root: id, Relation: Node{"version"}, Target: Node{"version", pkg.Version.String()}},
		Tripple{Root: id, Relation: Node{"source"}, Target: PackageId(dist, suite, "source", source)},
		Tripple{Root: id, Relation: Node{"arch"}, Target: Node{"arch", pkg.Architecture.String()}},
	}

	e, err := mail.ParseAddress(pkg.Maintainer)
	if err == nil {
		ret = append(ret, Tripple{Root: id, Relation: Node{"maintainer"}, Target: Node{"maintainer", e.Address}})
	}

	for _, possi := range pkg.Depends.GetAllPossibilities() {
		ret = append(ret, Tripple{Root: id, Relation: Node{"depends"}, Target: PackageId(dist, suite, arch, possi.Name)})
	}
	return ret, nil
}

func WriteTripples(w io.Writer, dist, suite, arch string) error {
	resp, err := http.Get(
		fmt.Sprintf("http://archive.paultag.house/debian/dists/%s/%s/%s/Packages.xz",
			dist, suite, arch,
		),
	)
	if err != nil {
		return err
	}

	reader, err := xz.NewReader(resp.Body, 0)
	if err != nil {
		return err
	}

	packages, err := archive.LoadPackages(reader)
	if err != nil {
		return err
	}

	for {
		pkg, err := packages.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		tripples, err := PackageTripples(dist, suite, arch, pkg)
		if err != nil {
			return err
		}
		for _, tripple := range tripples {
			fmt.Fprintf(w, "%s\n", tripple)
		}
	}
	return nil
}

func main() {
	out, err := os.Create("tripples.nq")
	ohshit(err)
	_ = out
	for _, arch := range []string{
		"binary-all",
		"binary-amd64",
		"binary-armhf",
		"binary-i386",
	} {
		ohshit(WriteTripples(out, "unstable", "main", arch))
		// WriteTripples(os.Stdout, "unstable", "main", arch)
	}
}
