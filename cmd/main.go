package main

import (
	"flag"
	"io/ioutil"

	"github.com/jmarthernandez/chip8"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	pathPtr := flag.String("path", "roms/invaders", "path to ROM")
	flag.Parse()
	rom, err := ioutil.ReadFile(*pathPtr)
	check(err)
	cpu := chip8.NewCPU()
	cpu.LoadRom(rom)
}
