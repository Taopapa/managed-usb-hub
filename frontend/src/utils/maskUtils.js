export const parseMaskToStates = (mask) => {
    if (mask === "FFFFFFFF") return Array(16).fill(true)
    if (mask === "00000000") return Array(16).fill(false)
    
    const states = Array(16).fill(false)
    if (!mask || mask.length !== 8) return states
    
    const byte0Hex = mask.substring(0, 2)
    const val = parseInt(byte0Hex, 16)
    if (isNaN(val)) return states
    
    for (let i = 0; i < 7; i++) {
        if ((val >> i) & 1) {
            states[i] = true
        }
    }
    return states
}

export const statesToMask = (states) => {
    // Check for All On
    if (states.every(s => s)) return "FFFFFFFF"
    // Check for All Off
    if (states.every(s => !s)) return "00000000"
    
    let byte0 = 0x80 // Bit 7 always set?
    for (let i = 0; i < 7; i++) {
        if (states[i]) byte0 |= (1 << i)
    }
    
    return byte0.toString(16).toUpperCase().padStart(2, '0') + "FFFFFF"
}
