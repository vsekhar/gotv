package tuner

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/dustin/go-humanize"
)

const fepath = "/dev/dvb/adapter0"

var stations map[string]Station = map[string]Station{
	"INVALID": {858000000, 1, 1, 1},
	"KGO-HD":  {177000000, 49, 52, 3},
	"KQED-HD": {569000000, 49, 51, 1},
}

const GOODSTATION = "KQED-HD"

func logStatus(tuner *Tuner, t *testing.T) {
	status, err := tuner.getStatus()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n", *status)
}

func logParams(tuner *Tuner, t *testing.T) {
	params, err := tuner.getParams()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n", *params)
}

func openFirstTunerOrDie(t *testing.T, path string, s Station) *Tuner {
	tuner, err := Open(path, s)
	if err != nil {
		t.Fatal(err)
	}
	return tuner
}

func TestTunerOpen(t *testing.T) {
	tuner := openFirstTunerOrDie(t, fepath, stations[GOODSTATION])
	defer tuner.Close()
	if tuner.fe.Name() != path.Join(fepath, "frontend0") {
		t.Errorf("tuner at '%s', expected '%s'", tuner.fe.Name(), fepath)
	}
	if tuner.fe.Fd() <= 2 {
		t.Errorf("tuner at unexpected fileno '%d'", tuner.fe.Fd())
	}
	inf, err := tuner.getInfo()
	if err != nil {
		t.Error(err)
	}
	if (inf.caps & FE_IS_STUPID) != 0 {
		t.Error("FE reports stupid")
	}
	if (inf.caps & FE_CAN_INVERSION_AUTO) == 0 {
		t.Error("FE cannot do auto inversion (this version assumes it can)")
	}
	t.Log(inf)
}

func TestTune(t *testing.T) {
	tuner := openFirstTunerOrDie(t, fepath, stations[GOODSTATION])
	defer tuner.Close()

	// confirm lock
	s, err := tuner.getStatus()
	if err != nil {
		t.Error(err)
	}
	if (*s & FE_HAS_LOCK) != FE_HAS_LOCK {
		t.Fatal("FE doesn't have lock")
	}
	t.Logf("tuned to %s", GOODSTATION)

	// dump some data
	tfile, err := ioutil.TempFile("", "gotv_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tfile.Name())
	t.Logf("starting capture to %s", tfile.Name())
	defer tfile.Close()

	var megs int64
	if testing.Short() {
		megs = 1
	} else {
		megs = 25
	}
	n, err := io.CopyN(tfile, tuner, megs*1024*1024)
	if err != nil {
		t.Error(err)
	}
	t.Logf("wrote %s bytes to %s", humanize.Bytes(uint64(n)), tfile.Name())
}

func TestTuneNoLock(t *testing.T) {
	tuner := openFirstTunerOrDie(t, fepath, stations["INVALID"])
	defer tuner.Close()

	// check for no lock
	s, err := tuner.getStatus()
	if err != nil {
		t.Fatal(err)
	}
	if (*s & FE_HAS_LOCK) == FE_HAS_LOCK {
		t.Fatalf("FE has lock, expected none")
	}
}
