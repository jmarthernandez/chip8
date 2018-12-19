package chip8

import (
	"fmt"
	"math/rand"
)

/*
CPU contains state of the emulated machine
+---------------+= 0xFFF (4095) End of Chip-8 RAM
|               |
|               |
|               |
|               |
|               |
| 0x200 to 0xFFF|
|     Chip-8    |
| Program / Data|
|     Space     |
|               |
|               |
|               |
+- - - - - - - -+= 0x600 (1536) Start of ETI 660 Chip-8 programs
|               |
|               |
|               |
+---------------+= 0x200 (512) Start of most Chip-8 programs
| 0x000 to 0x1FF|
| Reserved for  |
|  interpreter  |
+---------------+= 0x000 (0) Start of Chip-8 RAM
*/
type CPU struct {
	Opcode uint16
	Memory [4096]byte
	V      [16]byte
	I      uint16
	PC     uint16
	SP     byte
	DT     byte
	ST     byte
	Stack  [16]uint16
}

// NewCPU returns a new CPU struct with default options and loads the fontset
// into memory
func NewCPU() CPU {
	var cpu = CPU{
		PC:     0x200,
		Opcode: 0,
		I:      0,
		SP:     0,
	}

	for i, b := range FontSet {
		cpu.Memory[i] = b
	}

	return cpu
}

// LoadRom empties memory and loads program into memory
func (c *CPU) LoadRom(r []byte) {
	// Zero out memory after chip8 specific memory(fonts)
	for m := 512; m < 4096; m++ {
		c.Memory[m] = 0x00
	}

	// Loads ROM in byte by byte in to memory starting at 0x200
	for i, b := range r {
		c.Memory[i+0x200] = b
	}
}

/*
SetOpcode returns the next opcode to be performed
Chip8 opcodes are 16 bits meaining we need two consecutive bytes from memory

First we cast both bytes to unint16 so we have 16 bits to operate on

Assuming the following
c.Memory[c.PC]     = 0xA2   or 10100010
c.Memory[c.PC + 1] = 0xF0   or 11110000

After casting to uint16, notice the 8 bits added to the beginning
c.Memory[c.PC]     = 0x00A2 or 0000000010100010
c.Memory[c.PC + 1] = 0x00F0 or 0000000011110000

Now the confusing part.  We shift c.Memory[c.PC] 8 bits to the right
uint16(c.Memory[c.PC])<<8
0000000010100010 turns into 1010001000000000

Now using bitwise OR(|) we can effectively glue these two bytes together
c.Memory[c.PC]     1010001000000000 |
c.Memory[c.PC + 1] 0000000011110000 =
                   ----------------
                   1010001011110000

We then increment PC(program counter) by 2 to setup reading the next byte
*/
func (c *CPU) SetOpcode() {
	c.Opcode = uint16(c.Memory[c.PC])<<8 | uint16(c.Memory[c.PC+1])
}

/*
ExecuteOpcode excutes the opcode at c.Opcode
Chip8 opcodes are 16 bits but the instruction is only the first nibble(4 bits)
We can get this using the bitwise AND operator

Assume the following
c.Opcode = 1010001011110000 or 0xA2F0

1010001000000000 &
1111000000000000 =
----------------
1010000000000000

Reading this operation in hex is bit easier

0xA2F0 &
0xF000 =
------
0xA000

Conventions
nnn or addr - A 12-bit value, the lowest 12 bits of the instruction
n or nibble - A 4-bit value, the lowest 4 bits of the instruction
x - A 4-bit value, the lower 4 bits of the high byte of the instruction
y - A 4-bit value, the upper 4 bits of the low byte of the instruction
kk or byte - An 8-bit value, the lowest 8 bits of the instruction
*/
func (c *CPU) ExecuteOpcode() {
	switch c.Opcode & 0xF000 {
	case 0x0000:
		switch c.Opcode {
		case 0x00E0:
			// Clear the display.
			fmt.Printf("Not Implemented [0x0000]: 0x%X\n", c.Opcode)
			//  _            _
			// | |_ ___   __| | ___
			// | __/ _ \ / _` |/ _ \
			// | || (_) | (_| | (_) |
			//  \__\___/ \__,_|\___/
			c.PC += 2
			break
		case 0x00EE:
			// Return from a subroutine.
			// The interpreter sets the program counter to the address at the
			// top of the stack, then subtracts 1 from the stack pointer.
			c.PC = c.Stack[c.SP]
			c.SP--
			break
		default:
			fmt.Printf("Unknown opcode [0x0000]: 0x%X\n", c.Opcode)
			break
		}
	case 0x1000:
		// Jump to location nnn.
		// The interpreter sets the program counter to nnn.
		c.PC = c.Opcode & 0x0FFF
		break
	case 0x2000:
		// Call subroutine at nnn.
		// The interpreter increments the stack pointer, then puts the current
		// PC on the top of the stack. The PC is then set to nnn.
		c.SP++
		c.Stack[c.SP] = c.PC
		c.PC = c.Opcode & 0x0FFF
		break
	case 0x3000:
		// Skip next instruction if Vx = kk.
		// The interpreter compares register Vx to kk, and if they are equal,
		// increments the program counter by 2.
		vx := c.V[xNib(c.Opcode)]
		// casting to byte removes most significant(first) byte
		// byte(0x3411) => 0x11
		kk := byte(c.Opcode)
		c.PC += skipIf(vx == kk)
		break
	case 0x4000:
		// Skip next instruction if Vx != kk.
		// The interpreter compares register Vx to kk, and if they are not equal,
		// increments the program counter by 2.
		vx := c.V[xNib(c.Opcode)]
		kk := byte(c.Opcode)
		c.PC += skipIf(vx != kk)
		break
	case 0x5000:
		switch c.Opcode & 0xF00F {
		case 0x5000:
			// Skip next instruction if Vx = Vy.
			// The interpreter compares register Vx to register Vy, and if they
			// are equal, increments the program counter by 2.
			vx := c.V[xNib(c.Opcode)]
			vy := c.V[yNib(c.Opcode)]
			c.PC += skipIf(vx == vy)
			break
		default:
			fmt.Printf("Unknown opcode [0x0000]: 0x%X\n", c.Opcode)
			break
		}
	case 0x6000:
		// Set Vx = kk.
		// The interpreter puts the value kk into register Vx.
		x := xNib(c.Opcode)
		kk := byte(c.Opcode)
		c.V[x] = kk
		c.PC += 2
		break
	case 0x7000:
		// Set Vx = Vx + kk.
		// Adds the value kk to the value of register Vx,then stores the result in Vx.
		x := xNib(c.Opcode)
		kk := byte(c.Opcode)
		c.V[x] = c.V[x] + kk
		c.PC += 2
		break
	case 0x8000:
		x := xNib(c.Opcode)
		y := yNib(c.Opcode)
		switch c.Opcode & 0xF00F {
		case 0x8000:
			// Set Vx = Vy.
			// Stores the value of register Vy in register Vx.
			c.V[x] = c.V[y]
			c.PC += 2
			break
		case 0x8001:
			// Performs a bitwise OR on the values of Vx and Vy, then stores the result
			// in Vx. A bitwise OR compares the corrseponding bits from two values, and
			// if either bit is 1, then the same bit in the result is also 1. Otherwise, it is 0.
			c.V[x] = c.V[x] | c.V[y]
			c.PC += 2
			break
		case 0x8002:
			// Performs a bitwise AND on the values of Vx and Vy, then stores the result
			// in Vx. A bitwise AND compares the corrseponding bits from two values, and
			// if both bits are 1, then the same bit in the result is also 1. Otherwise, it is 0.
			c.V[x] = c.V[x] & c.V[y]
			c.PC += 2
			break
		case 0x8003:
			// Performs a bitwise exclusive OR on the values of Vx and Vy, then stores the
			// result in Vx. An exclusive OR compares the corrseponding bits from two values,
			// and if the bits are not both the same, then the corresponding bit in the
			// result is set to 1. Otherwise, it is 0.
			c.V[x] = c.V[x] ^ c.V[y]
			c.PC += 2
			break
		case 0x8004:
			// Set Vx = Vx + Vy, set VF = carry.
			// The values of Vx and Vy are added together. If the result is greater
			// than 8 bits (i.e., > 255,) VF is set to 1, otherwise 0. Only the lowest
			// 8 bits of the result are kept, and stored in Vx.
			c.V[x] = c.V[x] + c.V[y]
			c.V[0xF] = ternary(c.V[x] > 255)
			c.PC += 2
			break
		case 0x8005:
			// Set Vx = Vx - Vy, set VF = NOT borrow.
			// If Vx > Vy, then VF is set to 1, otherwise 0. Then Vy is subtracted
			// from Vx, and the results stored in Vx.
			c.V[x] = c.V[x] - c.V[y]
			c.V[0xF] = ternary(c.V[x] > c.V[y])
			c.PC += 2
			break
		case 0x8006:
			// Set Vx = Vx SHR 1.
			// If the least-significant bit of Vx is 1, then VF is set to 1,
			// otherwise 0. Then Vx is divided by 2.
			c.V[0xF] = ternary((c.V[x] & 0x0F) == 0x01)
			c.V[x] = c.V[x] / 2
			c.PC += 2
			break
		case 0x8007:
			// Set Vx = Vy - Vx, set VF = NOT borrow.
			// If Vy > Vx, then VF is set to 1, otherwise 0. Then Vx is
			// subtracted from Vy, and the results stored in Vx.
			c.V[x] = c.V[y] - c.V[x]
			c.V[0xF] = ternary(c.V[y] > c.V[x])
			c.PC += 2
			break
		case 0x800E:
			// Set Vx = Vx SHL 1.
			// If the least-significant bit of Vx is 1, then VF is set to 1,
			// otherwise 0. Then Vx is divided by 2.
			c.V[0xF] = ternary((c.V[x] & 0x0F) == 0x01)
			c.V[x] = c.V[x] * 2
			c.PC += 2
			break
		default:
			fmt.Printf("Unknown opcode [0x0000]: 0x%X\n", c.Opcode)
			break
		}
		break
	case 0x9000:
		switch c.Opcode & 0xF00F {
		case 0x9000:
			// Skip next instruction if Vx != Vy.
			// The values of Vx and Vy are compared, and if they are not equal, the program counter is increased by 2.
			vx := c.V[xNib(c.Opcode)]
			vy := c.V[yNib(c.Opcode)]
			c.PC += skipIf(vx != vy)
			break
		default:
			fmt.Printf("Unknown opcode [0x0000]: 0x%X\n", c.Opcode)
			break
		}
		break
	case 0xA000:
		// Set I = nnn.
		// The value of register I is set to nnn.
		c.I = c.Opcode & 0x0FFF
		c.PC += 2
		break
	case 0xB000:
		// Jump to location nnn + V0.
		// The program counter is set to nnn plus the value of V0.
		c.PC = uint16(c.V[0]) + (c.Opcode & 0x0FFF)
		break
	case 0xC000:
		// The interpreter generates a random number from 0 to 255, which is
		// then ANDed with the value kk. The results are stored in Vx. See
		// instruction 8xy2 for more information on AND.
		x := xNib(c.Opcode)
		kk := byte(c.Opcode)
		r := byte(rand.Intn(255))
		c.V[x] = kk & r
		c.PC += 2
		break
	case 0xD000:
		// Display n-byte sprite starting at memory location I at (Vx, Vy), set VF = collision.
		// The interpreter reads n bytes from memory, starting at the address stored in I.
		// These bytes are then displayed as sprites on screen at coordinates (Vx, Vy).
		// Sprites are XORed onto the existing screen. If this causes any pixels to be erased,
		// VF is set to 1, otherwise it is set to 0. If the sprite is positioned so
		// part of it is outside the coordinates of the display, it wraps around to
		// the opposite side of the screen. See instruction 8xy3 for more information on XOR

		// x := xNib(c.Opcode)
		// y := yNib(c.Opcode)
		// n := c.Opcode & 0x000F
		// sprites := c.Memory[c.I : c.I+n]

		fmt.Printf("Not Implemented [0x0000]: 0x%X\n", c.Opcode)
		//  _            _
		// | |_ ___   __| | ___
		// | __/ _ \ / _` |/ _ \
		// | || (_) | (_| | (_) |
		//  \__\___/ \__,_|\___/
		break
	case 0xE000:
		switch c.Opcode & 0xF0FF {
		case 0xE09E:
			// Skip next instruction if key with the value of Vx is pressed.
			// Checks the keyboard, and if the key corresponding to the value of
			// Vx is currently in the down position, PC is increased by 2.
			fmt.Printf("Not Implemented [0x0000]: 0x%X\n", c.Opcode)
			//  _            _
			// | |_ ___   __| | ___
			// | __/ _ \ / _` |/ _ \
			// | || (_) | (_| | (_) |
			//  \__\___/ \__,_|\___/
			break
		case 0xE0A1:
			fmt.Printf("Not Implemented [0x0000]: 0x%X\n", c.Opcode)
			//  _            _
			// | |_ ___   __| | ___
			// | __/ _ \ / _` |/ _ \
			// | || (_) | (_| | (_) |
			//  \__\___/ \__,_|\___/
			break
		default:
			fmt.Printf("Unknown opcode [0x0000]: 0x%X\n", c.Opcode)
			break
		}
	case 0xF000:
		x := xNib(c.Opcode)
		switch c.Opcode & 0xF0FF {
		case 0xF007:
			// Set Vx = delay timer value.
			// The value of DT is placed into Vx.
			c.V[x] = c.DT
			c.PC += 2
			break
		case 0xF00A:
			// Wait for a key press, store the value of the key in Vx.
			// All execution stops until a key is pressed, then the value of that key is stored in Vx.
			fmt.Printf("Not Implemented [0x0000]: 0x%X\n", c.Opcode)
			//  _            _
			// | |_ ___   __| | ___
			// | __/ _ \ / _` |/ _ \
			// | || (_) | (_| | (_) |
			//  \__\___/ \__,_|\___/
			break
		case 0xF015:
			// Set delay timer = Vx.
			// DT is set equal to the value of Vx.
			c.DT = c.V[x]
			c.PC += 2
			break
		case 0xF018:
			// Set sound timer = Vx.
			// ST is set equal to the value of Vx.
			c.ST = c.V[x]
			c.PC += 2
			break
		case 0xF01E:
			// Set I = I + Vx.
			// The values of I and Vx are added, and the results are stored in I.
			c.I = c.I + uint16(c.V[x])
			c.PC += 2
			break
		case 0xF029:
			// Set I = location of sprite for digit Vx.
			// The value of I is set to the location for the hexadecimal sprite
			// corresponding to the value of Vx.
			fmt.Printf("Not Implemented [0x0000]: 0x%X\n", c.Opcode)
			//  _            _
			// | |_ ___   __| | ___
			// | __/ _ \ / _` |/ _ \
			// | || (_) | (_| | (_) |
			//  \__\___/ \__,_|\___/
			break
		case 0xF033:
			// Store BCD representation of Vx in memory locations I, I+1, and I+2.
			// The interpreter takes the decimal value of Vx, and places the hundreds
			// digit in memory at location in I, the tens digit at location I+1,
			// and the ones digit at location I+2.
			fmt.Printf("Not Implemented [0x0000]: 0x%X\n", c.Opcode)
			//  _            _
			// | |_ ___   __| | ___
			// | __/ _ \ / _` |/ _ \
			// | || (_) | (_| | (_) |
			//  \__\___/ \__,_|\___/
			break
		case 0xF055:
			// Store registers V0 through Vx in memory starting at location I.
			// The interpreter copies the values of registers V0 through Vx into
			// memory, starting at the address in I.
			for i := uint16(0); i <= uint16(x); i++ {
				c.Memory[c.I+i] = c.V[i]
			}
			c.PC += 2
			break
		case 0xF065:
			// Read registers V0 through Vx from memory starting at location I.
			// The interpreter reads values from memory starting at location I
			// into registers V0 through Vx.
			for i := uint16(0); i <= uint16(x); i++ {
				c.V[i] = c.Memory[c.I+i]
			}
			break
		}
	default:
		fmt.Printf("Unknown opcode: 0x%X\n", c.Opcode)
	}
}

// skipIf takes an expression and returns a value to increment the PC by
// 2 - we move to the next opcode because they are two bytes long
// 4 - we skip the next opcode
func skipIf(exp bool) uint16 {
	if exp {
		return 4
	}
	return 2
}

func xNib(opcode uint16) uint16 {
	return (opcode & 0x0F00) >> 8
}

func yNib(opcode uint16) uint16 {
	return (opcode & 0x00F0) >> 4
}

func ternary(exp bool) byte {
	if exp {
		return 1
	}
	return 0
}
