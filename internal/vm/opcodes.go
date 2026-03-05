package vm

type Opcode byte

const (
    // Core 3D memory operations
    OP_NOP      Opcode = 0x00
    OP_MLOAD3D  Opcode = 0x10
    OP_MSTORE3D Opcode = 0x11
    OP_MALLOC3D Opcode = 0x12
    OP_MFREE3D  Opcode = 0x13
    OP_MCOPY3D  Opcode = 0x14
    OP_MSIZE3D  Opcode = 0x15
    
    // Spatial operations
    OP_MQUERY3D  Opcode = 0x16
    OP_MCOMPACT3D Opcode = 0x17
    OP_MDEFRAG3D Opcode = 0x18
    
    // Batch operations
    OP_MLOADBATCH3D  Opcode = 0x19
    OP_MSTOREBATCH3D Opcode = 0x1A
    
    // Gas operations
    OP_MGAS3D    Opcode = 0x1B
    OP_MREFUND3D Opcode = 0x1C
    
    // Control flow
    OP_JUMP3D    Opcode = 0x20
    OP_CALL3D    Opcode = 0x21
    OP_RET3D     Opcode = 0x22
    
    // Register operations
    OP_MOV3D     Opcode = 0x30
    OP_ADD3D     Opcode = 0x31
    OP_SUB3D     Opcode = 0x32
    
    // System operations
    OP_HALT3D    Opcode = 0xFF
)
