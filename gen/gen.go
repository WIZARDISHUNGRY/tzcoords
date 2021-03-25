package main

// @ +build ignore

import (
	"bufio"
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/WIZARDISHUNGRY/tzcoords/internal"
)

var output = flag.String("output", "", "output file name")

//go:embed zone1970.tab
var zone1970 []byte

type Generator struct {
	buf bytes.Buffer // Accumulated output.
}

func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

// format returns the gofmt-ed contents of the Generator's buffer.
func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		err := fmt.Errorf("warning: internal error: invalid Go generated: %w", err)
		panic(err)
	}
	return src
}

func main() {
	flag.Parse()
	outputName := *output
	if outputName == "" {
		flag.Usage()
		os.Exit(1)
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		panic("couldn't ReadBuildInfo")
	}
	internalModPath := bi.Main.Path + "/internal"

	const (
		coordsOffset = 1
		nameOffset   = 2
	)

	scanner := bufio.NewScanner(bytes.NewReader(zone1970))
	toCoords := make(map[string]internal.LatLon)
	keys := make([]string, 0)
	for scanner.Scan() {
		str := scanner.Text()
		parts := strings.Split(str, "\t")
		if str[0] == '#' || len(parts) <= nameOffset {
			continue
		}

		coords := parts[coordsOffset]
		name := parts[nameOffset]

		ll, err := internal.ParseISO6709Pair(coords)
		if err != nil {
			err := fmt.Errorf("parseISO6709Pair %s [%s]: %w", name, coords, err)
			panic(err)
		}

		if _, err := time.LoadLocation(name); err != nil {
			err := fmt.Errorf("LoadLocation %s: %w", name, err)
			panic(err)
		}
		toCoords[name] = ll
		keys = append(keys, name)
	}

	sort.Strings(keys)

	g := Generator{}

	g.Printf("// Code generated by \"go run ./gen/... %s\"; DO NOT EDIT.\n", strings.Join(os.Args[1:], " "))
	g.Printf("\n")
	g.Printf("package %s", "tzcoords")
	g.Printf("\n")
	g.Printf("import \"%s\"\n", internalModPath)
	g.Printf("\n")
	g.Printf("var toCoords = map[string]internal.LatLon{\n")
	for _, name := range keys {
		ll := toCoords[name]
		g.Printf("\t\"%s\": {Lat: %f, Lon: %f},\n", name, ll.Lat, ll.Lon)
	}
	g.Printf("}\n")
	g.Printf("\n")
	src := g.format()
	err := ioutil.WriteFile(outputName, src, 0644)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
}
