package vm

import (
	"encoding/binary"
	"fmt"

	"github.com/rawbytedev/memory3d/internal/types"
)

type OperandType byte

const (
	OT_IMMEDIATE OperandType = iota
	OT_REGISTER
	OT_ADDRESS
	OT_ADDRESS_OFFSET
	OT_SIZE
	OT_FLAGS
)

type Operand struct {
	Type  OperandType
	Value interface{}
}

type Instruction3D struct {
	Opcode   Opcode
	Operands []Operand
	Size     uint32
	PC       uint64 // Program counter value
	GasLimit uint64 // Max gas for this instruction
}

func DecodeInstruction(data []byte) (*Instruction3D, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("empty instruction")
	}

	opcode := Opcode(data[0])
	_ = &Instruction3D{
		Opcode: opcode,
		Size:   uint32(len(data)),
	}

	// Decode based on opcode
	switch opcode {
	case OP_MLOAD3D:
		return decodeMLoad3D(data)
	case OP_MSTORE3D:
		return decodeMStore3D(data)
	case OP_MALLOC3D:
		return decodeMAlloc3D(data)
	case OP_MFREE3D:
		return decodeMFree3D(data)
	case OP_MCOPY3D:
		return decodeMCopy3D(data)
	case OP_ADD3D:
		return decodeAdd3D(data)
	case OP_SUB3D:
		return decodeSub3D(data)
	case OP_MOV3D:
		return decodeMov3D(data)
	case OP_MSIZE3D:
		return decodeMSize3D(data)
	case OP_NOP:
		return &Instruction3D{Opcode: OP_NOP, Size: 1}, nil
	case OP_HALT3D:
		return &Instruction3D{Opcode: OP_HALT3D, Size: 1}, nil
	default:
		return nil, fmt.Errorf("unknown opcode: 0x%x", opcode)
	}
}

func decodeSub3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][destReg(1)][srcReg1(1)][srcReg2(1)]
	if len(data) != 4 {
		return nil, fmt.Errorf("SUB3D requires 4 bytes")
	}
	srcType1 := Register(data[2])
	srcType2 := Register(data[3])
	destType := Register(data[1])
	return &Instruction3D{
		Opcode: OP_SUB3D,
		Operands: []Operand{
			{Type: OT_REGISTER, Value: destType},
			{Type: OT_REGISTER, Value: srcType1},
			{Type: OT_REGISTER, Value: srcType2},
		},
		Size: 4,
	}, nil
}

func decodeAdd3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][destReg(1)][srcReg1(1)][srcReg2(1)]
	if len(data) != 4 {
		return nil, fmt.Errorf("ADD3D requires 4 bytes")
	}
	srcType1 := Register(data[2])
	srcType2 := Register(data[3])
	destType := Register(data[1])
	return &Instruction3D{
		Opcode: OP_ADD3D,
		Operands: []Operand{
			{Type: OT_REGISTER, Value: destType},
			{Type: OT_REGISTER, Value: srcType1},
			{Type: OT_REGISTER, Value: srcType2},
		},
		Size: 4,
	}, nil
}

func decodeMCopy3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][srcX(8)][srcY(2)][srcZ(2)][dstX(4)][dstY(2)][size(1)][padding(1)]
	if len(data) != 20 {
		return nil, fmt.Errorf("MCOPY3D requires 20 bytes")
	}

	srcX := binary.BigEndian.Uint64(data[1:9])
	srcY := binary.BigEndian.Uint16(data[9:11])
	srcZ := binary.BigEndian.Uint16(data[11:13])

	// Destination uses 4 bytes for X to fit in 20-byte limit
	dstX := uint64(binary.BigEndian.Uint32(data[13:17]))
	dstY := binary.BigEndian.Uint16(data[17:19])
	size := uint32(data[19])

	return &Instruction3D{
		Opcode: OP_MCOPY3D,
		Operands: []Operand{
			{Type: OT_ADDRESS, Value: types.Address3D{X: srcX, Y: uint32(srcY), Z: srcZ}},
			{Type: OT_ADDRESS, Value: types.Address3D{X: dstX, Y: uint32(dstY), Z: 0}},
			{Type: OT_SIZE, Value: size},
		},
		Size: 20,
	}, nil
}

func decodeMFree3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][x(8)][y(4)][z(2)]
	if len(data) != 15 {
		return nil, fmt.Errorf("MFREE3D requires 15 bytes")
	}

	x := binary.BigEndian.Uint64(data[1:9])
	y := binary.BigEndian.Uint32(data[9:13])
	z := binary.BigEndian.Uint16(data[13:15])

	return &Instruction3D{
		Opcode: OP_MFREE3D,
		Operands: []Operand{
			{Type: OT_ADDRESS, Value: types.Address3D{X: x, Y: y, Z: z}},
		},
		Size: 15,
	}, nil
}

func decodeMAlloc3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][size(4)][regionType(1)][flags(1)][destReg(1)][padding(6)]
	if len(data) != 14 {
		return nil, fmt.Errorf("MALLOC3D requires 14 bytes")
	}

	size := binary.BigEndian.Uint32(data[1:5])
	regionType := types.RegionType(data[5])
	flags := types.AllocationFlags(data[6])
	// destReg is data[7], usually R0 by convention, but can be used for future extensions

	return &Instruction3D{
		Opcode: OP_MALLOC3D,
		Operands: []Operand{
			{Type: OT_SIZE, Value: size},
			{Type: OT_FLAGS, Value: regionType},
			{Type: OT_FLAGS, Value: flags},
		},
		Size: 14,
	}, nil
}

func decodeMLoad3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][x(8)][y(4)][z(2)][size(4)][reg(1)]
	if len(data) != 20 {
		return nil, fmt.Errorf("MLOAD3D requires 20 bytes")
	}

	x := binary.BigEndian.Uint64(data[1:9])
	y := binary.BigEndian.Uint32(data[9:13])
	z := binary.BigEndian.Uint16(data[13:15])
	size := binary.BigEndian.Uint32(data[15:19])
	reg := Register(data[19])

	return &Instruction3D{
		Opcode: OP_MLOAD3D,
		Operands: []Operand{
			{Type: OT_ADDRESS, Value: types.Address3D{X: x, Y: y, Z: z}},
			{Type: OT_SIZE, Value: size},
			{Type: OT_REGISTER, Value: reg},
		},
		Size: 20,
	}, nil
}

func decodeMStore3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][x(8)][y(4)][z(2)][reg(1)][size(4)]
	if len(data) != 20 {
		return nil, fmt.Errorf("MSTORE3D requires 20 bytes")
	}

	x := binary.BigEndian.Uint64(data[1:9])
	y := binary.BigEndian.Uint32(data[9:13])
	z := binary.BigEndian.Uint16(data[13:15])
	reg := Register(data[15])
	size := binary.BigEndian.Uint32(data[16:20])

	return &Instruction3D{
		Opcode: OP_MSTORE3D,
		Operands: []Operand{
			{Type: OT_ADDRESS, Value: types.Address3D{X: x, Y: y, Z: z}},
			{Type: OT_REGISTER, Value: reg},
			{Type: OT_SIZE, Value: size},
		},
		Size: 20,
	}, nil
}
func decodeMov3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][destReg(1)][srcReg(1)][padding(5)]
	if len(data) != 8 {
		return nil, fmt.Errorf("MOV3D requires 8 bytes")
	}

	destReg := Register(data[1])
	srcReg := Register(data[2])

	return &Instruction3D{
		Opcode: OP_MOV3D,
		Operands: []Operand{
			{Type: OT_REGISTER, Value: destReg},
			{Type: OT_REGISTER, Value: srcReg},
		},
		Size: 8,
	}, nil
}

func decodeMSize3D(data []byte) (*Instruction3D, error) {
	// Format: [opcode(1)][x(8)][y(4)][z(2)][destReg(1)]
	if len(data) != 16 {
		return nil, fmt.Errorf("MSIZE3D requires 16 bytes")
	}

	x := binary.BigEndian.Uint64(data[1:9])
	y := binary.BigEndian.Uint32(data[9:13])
	z := binary.BigEndian.Uint16(data[13:15])
	destReg := Register(data[15])

	return &Instruction3D{
		Opcode: OP_MSIZE3D,
		Operands: []Operand{
			{Type: OT_ADDRESS, Value: types.Address3D{X: x, Y: y, Z: z}},
			{Type: OT_REGISTER, Value: destReg},
		},
		Size: 16,
	}, nil
}