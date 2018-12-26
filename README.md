# chip8
chip8 emulator in go

## Usage

```
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

```

Example usage can be found in the [cmd](cmd/main.go) directory

### TODO
- opcodes that wait on key events
- opcodes that interact with graphics
- timers
- pluggable graphics

## Sources
Most of the comments are some combination of information I gathered from the following websites

- http://www.multigesture.net/articles/how-to-write-an-emulator-chip-8-interpreter/
- http://emulator101.com/
- http://devernay.free.fr/hacks/chip8/C8TECH10.HTM#2.4

## ROMs
The roms included are in the public domain and were downloaded from https://www.zophar.net/pdroms/chip8/chip-8-games-pack.html
