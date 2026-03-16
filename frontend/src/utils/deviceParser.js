function hexToAscii(hex) {
    let str = '';
    for (let i = 0; i < hex.length; i += 2) {
        const code = parseInt(hex.substr(i, 2), 16);
        if (!isNaN(code)) str += String.fromCharCode(code);
    }
    return str;
}

export const parseDevice = (r) => {
    // Parse Version
    let version = "v1.0"
    if (r.probeResponse) {
        const ascii = r.asciiResponse
        const vIndex = ascii.indexOf('v')
        if (vIndex !== -1) {
            let vEnd = ascii.indexOf('\r', vIndex)
            if (vEnd === -1) vEnd = ascii.indexOf('\n', vIndex)
            if (vEnd === -1) vEnd = ascii.length
            version = ascii.substring(vIndex, vEnd).trim()
        }
    }

    // Parse GO Data
    let totalPorts = 8
    let validPortsMask = 0xFFFF
    if (r.goData) {
        let goAscii = ""
        try {
            const asciiStr = r.goData.substring(0, 4)
            goAscii = hexToAscii(asciiStr)
        } catch(e) {}
        
        let maskVal = parseInt(goAscii, 16)
        if (!isNaN(maskVal)) {
            validPortsMask = maskVal
            let count = 0
            let temp = maskVal
            while (temp > 0) {
                if (temp & 1) count++
                temp >>= 1
            }
            totalPorts = count
        }
    }

    // Parse GP Data
    let onPortsList = []
    let offPortsList = []
    if (r.ledStatus) {
        let hexStr = r.ledStatus
        let asciiStr = ""
        try {
            for (let i = 0; i < hexStr.length; i += 2) {
                const code = parseInt(hexStr.substr(i, 2), 16)
                if (!isNaN(code)) asciiStr += String.fromCharCode(code)
            }
        } catch(e) {}
        
        let isAllOn = false
        let isAllOff = false
        
        // Clean up asciiStr
        asciiStr = asciiStr.replace(/[\r\n\s]+/g, '').trim()
        
        // Remove prefixes like 'GP', 'G', 'g', 'gp'
        if (asciiStr.toUpperCase().startsWith("GP")) asciiStr = asciiStr.substring(2)
        else if (asciiStr.toUpperCase().startsWith("G")) asciiStr = asciiStr.substring(1)
        
        if (asciiStr.length >= 2) {
            const prefix = asciiStr.substring(0, 2).toUpperCase()
            if (prefix === "FF") isAllOn = true
            else if (prefix === "00") isAllOff = true
        }
        
        if (isAllOn) {
            for (let i = 1; i <= totalPorts; i++) onPortsList.push(i)
        } else if (isAllOff) {
            for (let i = 1; i <= totalPorts; i++) offPortsList.push(i)
        } else {
            // Try to find a valid hex pattern
            // Look for 8 chars first, then 2 chars
            let match = asciiStr.match(/[0-9A-Fa-f]{8}/)
            if (!match) match = asciiStr.match(/[0-9A-Fa-f]{2}/)
            
            if (match) {
                const hexVal = match[0]
                const val = parseInt(hexVal.substring(0, 2), 16)
                if (!isNaN(val)) {
                     for (let i = 1; i <= totalPorts; i++) {
                         if (i <= 7) {
                             if ((val >> (i - 1)) & 1) {
                                 onPortsList.push(i)
                             } else {
                                 offPortsList.push(i)
                             }
                         } else {
                             offPortsList.push(i)
                         }
                     }
                }
            } else {
                 for (let i = 1; i <= totalPorts; i++) offPortsList.push(i)
            }
        }
    }

    return {
        portId: r.path,
        portName: r.path, // e.g., COM5
        displayName: `C2G54464 7-Port Hub (${r.path})`, // Mock display name
        description: 'C2G 7-Port Usb-A Hub',
        firmwareVersion: version,
        totalPorts: totalPorts,
        onPortsDisplay: onPortsList.length > 0 ? onPortsList.join(',') : '',
        offPortsDisplay: offPortsList.length > 0 ? offPortsList.join(',') : ''
    }
}
