import { CONSTANTS } from '../config/constants'

function hexToAscii(hex) {
    let str = '';
    for (let i = 0; i < hex.length; i += 2) {
        const code = parseInt(hex.substr(i, 2), 16);
        if (!isNaN(code)) str += String.fromCharCode(code);
    }
    return str;
}

export const parseDevice = (r) => {
    const deviceName = (r.deviceName || '').replace(/[^\x20-\x7E]/g, '').trimEnd()

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

    // Currently, this product only has 7 ports. Hardcoding to 7.
    let totalPorts = 7

    // Parse GW Data
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
        
        // Remove prefixes like 'GW', 'G', 'g'
        if (asciiStr.toUpperCase().startsWith("GW")) asciiStr = asciiStr.substring(2)
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
                         if (i <= totalPorts) { // Use totalPorts dynamically instead of hardcoded 7
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
        deviceName: deviceName,
        displayName: deviceName ? `${deviceName} (${r.path})` : `C2G 7-port Managed USB HUB (${r.path})`,
        description: 'C2G 7-port Managed USB HUB',
        firmwareVersion: version,
        totalPorts: totalPorts,
        onPortsDisplay: onPortsList.length > 0 ? onPortsList.join(',') : '',
        offPortsDisplay: offPortsList.length > 0 ? offPortsList.join(',') : ''
    }
}
