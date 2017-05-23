package main

import (
	"path"
	"sort"
)

type IOs struct {
	Cnt int // IOs reference count

	Map map[bool]map[int]bool // input(false)/output(true) inodes
}

type Cmd struct {
	Parent int // parent Cmd ID
	ID     int // Cmd ID, changes only with execve

	Dir  string
	Path string
	Args []string
	Env  []string `json:",omitempty"`
}

type ProcState struct {
	SysEnter bool        // true on enter to syscall
	Syscall  int         // call number on exit from syscall
	CurDir   string      // working directory
	CurCmd   Cmd         // current command
	NextCmd  Cmd         // command after return from execve
	FDs      map[int]int // map fds to inodes

	IOs *IOs
}

type Record struct {
	Cmd     Cmd
	Inputs  []string
	Outputs []string
	FDs     map[int]string
}

func NewIOs() *IOs {
	return &IOs{1, map[bool]map[int]bool{
		false: make(map[int]bool),
		true:  make(map[int]bool),
	}}
}

func NewProcState() *ProcState {
	return &ProcState{
		FDs: make(map[int]int),
		IOs: NewIOs(),
	}
}

func (ps *ProcState) ResetIOs() {
	ps.IOs.Cnt--
	ps.IOs = NewIOs()
}

func (ps *ProcState) Abs(p string) string {
	return ps.AbsAt(ps.CurDir, p)
}

func (ps *ProcState) AbsAt(dir, p string) string {
	if !path.IsAbs(p) {
		p = path.Join(dir, p)
	}
	return path.Clean(p)
}

func (ps *ProcState) Clone() *ProcState {
	newps := NewProcState()
	newps.IOs = ps.IOs // IOs are shared until exec
	ps.IOs.Cnt++
	newps.CurDir = ps.CurDir
	newps.CurCmd = ps.CurCmd
	for n, s := range ps.FDs {
		newps.FDs[n] = s
	}
	return newps
}

func (ps *ProcState) Record(sys *SysState) Record {
	r := Record{Cmd: ps.CurCmd, Inputs: []string{}, Outputs: []string{}}
	for output, inodes := range ps.IOs.Map {
		// Deduplicate paths after renames.
		seen := map[string]bool{}
		paths := &r.Inputs
		if output {
			paths = &r.Outputs
		}
		for inode := range inodes {
			s := sys.FS.Path(inode)
			if !seen[s] {
				seen[s] = true
				*paths = append(*paths, s)
			}
		}
	}
	sort.Strings(r.Inputs)
	sort.Strings(r.Outputs)
	return r
}
