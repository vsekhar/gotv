// +build linux

package tuner

/*
	#include <linux/ioctl.h>
	#include <linux/dvb/frontend.h>
	#include <linux/dvb/dmx.h>

	// These are needed because Go cannot access constants defined via #DEFINE
	// (from: https://www.linuxtv.org/docs/dvbapi/DVB_Frontend_API.html)
	const int iFE_GET_INFO = FE_GET_INFO;
	const int iFE_READ_STATUS = FE_READ_STATUS;
	const int iFE_READ_BER = FE_READ_BER;
	const int iFE_READ_SNR = FE_READ_SNR;
	const int iFE_READ_SIGNAL_STRENGTH = FE_READ_SIGNAL_STRENGTH;
	const int iFE_SET_FRONTEND = FE_SET_FRONTEND;
	const int iFE_GET_FRONTEND = FE_GET_FRONTEND;
	const int iFE_GET_EVENT = FE_GET_EVENT;

	const int iDMX_IMMEDIATE_START = DMX_IMMEDIATE_START;
	const int iDMX_SET_PES_FILTER = DMX_SET_PES_FILTER;
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"
	"unsafe"
)

// Re-declared in Go so tests can access them
const (
	// fe_caps_t
	FE_IS_STUPID          = C.FE_IS_STUPID
	FE_CAN_INVERSION_AUTO = C.FE_CAN_INVERSION_AUTO

	// fe_status_t
	FE_HAS_SIGNAL  = C.FE_HAS_SIGNAL
	FE_HAS_CARRIER = C.FE_HAS_CARRIER
	FE_HAS_VITERBI = C.FE_HAS_VITERBI
	FE_HAS_SYNC    = C.FE_HAS_SYNC
	FE_HAS_LOCK    = C.FE_HAS_LOCK
	FE_TIMEDOUT    = C.FE_TIMEDOUT
)

type Station struct {
	Freq int
	Vid  int
	Aid  int
	Pid  int
}

type Tuner struct {
	path string
	fe   *os.File
	dmxv *os.File
	dmxa *os.File
	dmxd *os.File
	dvr  *os.File
}

// from: https://github.com/davecheney/pcap/blob/master/bpf.go#L43
func ioctl(fd uintptr, request C.int, argp unsafe.Pointer) error {
	_, _, errorp := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(request), uintptr(argp))
	if errorp != 0 {
		return os.NewSyscallError("ioctl", syscall.Errno(errorp))
	}
	return nil
}

func Open(p string, s Station) (*Tuner, error) {
	// Resource: https://www.linuxtv.org/docs/dvbapi/Examples.html

	// TODO: detect correct component numbers (assumed 0 here)
	tuner := new(Tuner)
	tuner.path = p

	// frontend
	fe, err := os.OpenFile(path.Join(p, "frontend0"), os.O_RDWR, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	tuner.fe = fe
	feparams := new(C.struct_dvb_frontend_parameters)
	feparams.frequency = C.__u32(s.Freq)
	feparams.inversion = C.INVERSION_AUTO
	dvbparams := (*C.struct_dvb_vsb_parameters)(unsafe.Pointer(&feparams.u))
	dvbparams.modulation = C.VSB_8
	err = ioctl(tuner.fe.Fd(), C.iFE_SET_FRONTEND, unsafe.Pointer(feparams))
	if err != nil {
		return nil, err
	}
	ev, err := tuner.getEvent()
	if ev.status == FE_TIMEDOUT {
		return nil, fmt.Errorf("tuner timed out while tuning to %d", s.Freq)
	}
	if int(ev.parameters.frequency) != s.Freq {
		return nil, fmt.Errorf("failed to tune to %d, tuner at %d", s.Freq, int(ev.parameters.frequency))
	}

	// demux: video
	dmxv, err := os.OpenFile(path.Join(p, "demux0"), os.O_RDWR, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	tuner.dmxv = dmxv
	pesparams := new(C.struct_dmx_pes_filter_params)
	pesparams.pid = C.__u16(s.Vid)
	pesparams.input = C.DMX_IN_FRONTEND
	pesparams.output = C.DMX_OUT_TS_TAP
	pesparams.pes_type = C.DMX_PES_VIDEO
	pesparams.flags = C.__u32(C.iDMX_IMMEDIATE_START)
	err = ioctl(tuner.dmxv.Fd(), C.iDMX_SET_PES_FILTER, unsafe.Pointer(pesparams))
	if err != nil {
		return nil, err
	}

	// demux: audio
	dmxa, err := os.OpenFile(path.Join(p, "demux0"), os.O_RDWR, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	tuner.dmxa = dmxa
	pesparams = new(C.struct_dmx_pes_filter_params)
	pesparams.pid = C.__u16(s.Aid)
	pesparams.input = C.DMX_IN_FRONTEND
	pesparams.output = C.DMX_OUT_TS_TAP
	pesparams.pes_type = C.DMX_PES_AUDIO
	pesparams.flags = C.__u32(C.iDMX_IMMEDIATE_START)
	err = ioctl(tuner.dmxa.Fd(), C.iDMX_SET_PES_FILTER, unsafe.Pointer(pesparams))
	if err != nil {
		return nil, err
	}

	// demux: data
	dmxd, err := os.OpenFile(path.Join(p, "demux0"), os.O_RDWR, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	tuner.dmxd = dmxd
	pesparams = new(C.struct_dmx_pes_filter_params)
	pesparams.pid = C.__u16(s.Pid)
	pesparams.input = C.DMX_IN_FRONTEND
	pesparams.output = C.DMX_OUT_TS_TAP
	pesparams.pes_type = C.DMX_PES_TELETEXT
	pesparams.flags = C.__u32(C.iDMX_IMMEDIATE_START)
	err = ioctl(tuner.dmxd.Fd(), C.iDMX_SET_PES_FILTER, unsafe.Pointer(pesparams))
	if err != nil {
		return nil, err
	}

	// dvr
	dvr, err := os.OpenFile(path.Join(p, "dvr0"), os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	tuner.dvr = dvr

	return tuner, nil
}

func (t *Tuner) Read(buf []byte) (int, error) {
	return t.dvr.Read(buf)
}

func (t *Tuner) Close() error {
	type closer_t func() error
	closers := []closer_t{
		t.fe.Close,
		t.dmxv.Close,
		t.dmxa.Close,
		t.dmxd.Close,
		t.dvr.Close,
	}
	ess := make([]string, 0)
	for _, closer := range closers {
		e := closer()
		if e != nil {
			ess = append(ess, e.Error())
		}
	}
	if len(ess) > 0 {
		return errors.New(strings.Join(ess, "; "))
	}
	return nil
}

func (t *Tuner) getParams() (*C.struct_dvb_frontend_parameters, error) {
	rparams := new(C.struct_dvb_frontend_parameters)
	err := ioctl(t.fe.Fd(), C.iFE_GET_FRONTEND, unsafe.Pointer(rparams))
	if err != nil {
		return nil, err
	}
	return rparams, nil
}

func (t *Tuner) getStatus() (*C.fe_status_t, error) {
	status := new(C.fe_status_t)
	err := ioctl(t.fe.Fd(), C.iFE_READ_STATUS, unsafe.Pointer(status))
	if err != nil {
		return nil, err
	}
	return status, nil
}

func (t *Tuner) getInfo() (*C.struct_dvb_frontend_info, error) {
	dfi := new(C.struct_dvb_frontend_info)
	err := ioctl(t.fe.Fd(), C.iFE_GET_INFO, unsafe.Pointer(dfi))
	if err != nil {
		return nil, err
	}
	return dfi, nil
}

// Block until an event arrives
func (t *Tuner) getEvent() (*C.struct_dvb_frontend_event, error) {
	ev := new(C.struct_dvb_frontend_event)
	err := ioctl(t.fe.Fd(), C.iFE_GET_EVENT, unsafe.Pointer(ev))
	if err != nil {
		return nil, err
	}
	return ev, nil
}

// For tests/logging
func (inf *C.struct_dvb_frontend_info) String() string {
	steps := (int(inf.frequency_max) - int(inf.frequency_min)) / int(inf.frequency_stepsize)
	return fmt.Sprintf("Device name: %s\n"+
		"Min Freq: %d\n"+
		"Max Freq: %d\n"+
		"Stepsize: %d (%d steps)\n"+
		"Capabilities: %#0x",
		C.GoString(&inf.name[0]),
		int(inf.frequency_min),
		int(inf.frequency_max),
		int(inf.frequency_stepsize), steps,
		inf.caps)
}

func (params *C.struct_dvb_frontend_parameters) String() string {
	return fmt.Sprintf("")
}
