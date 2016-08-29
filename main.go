package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/vsekhar/gotv/tuner"
)

var outputfile = flag.String("out", "", "output file")
var size = flag.Int64("size", 1, "amount of data to capture in MB")

func main() {
	flag.Parse()

	s := tuner.Station{177000000, 49, 52, 3} // KQED-HD in SF
	tv, err := tuner.Open("/dev/dvb/adapter0", s)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	ofile, err := os.Create(*outputfile)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer ofile.Close()
	log.Printf("starting capture to %s", ofile.Name())

	log.Println("starting copy")
	io.CopyN(ofile, tv, (*size)*1024*1024)
	log.Println("done copy")
}
