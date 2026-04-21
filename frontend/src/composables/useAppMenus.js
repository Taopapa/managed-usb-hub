import { computed } from 'vue'

export const useAppMenus = ({
    currentDevice,
    autoSearch,
    exportLogs,
    quitApp,
    refreshHub,
    openDeviceNameModal,
    openDeviceUidModal,
    openVBUSPowerModal,
    runCli,
    openDocumentationModal,
    showAbout
}) => {
    return computed(() => [
        {
            key: 'file',
            label: 'File',
            items: [
                { key: 'scan', label: 'Scan for USB Hubs', onClick: autoSearch },
                { key: 'sep-1', separator: true },
                { key: 'export-logs', label: 'Export Logs', onClick: exportLogs },
                { key: 'sep-2', separator: true },
                { key: 'exit', label: 'Exit', onClick: quitApp }
            ]
        },
        {
            key: 'view',
            label: 'View',
            items: [
                {
                    key: 'refresh-hub',
                    label: 'Refresh Selected Hub',
                    onClick: refreshHub,
                    disabled: !currentDevice.value
                }
            ]
        },
        {
            key: 'tools',
            label: 'Tools',
            items: [
                {
                    key: 'device-name',
                    label: 'Set Device Name',
                    onClick: openDeviceNameModal,
                    disabled: !currentDevice.value
                },
                { key: 'sep-1', separator: true },
                {
                    key: 'device-uid',
                    label: 'Set Device UID',
                    onClick: openDeviceUidModal,
                    disabled: !currentDevice.value
                },
                { key: 'sep-2', separator: true },
                {
                    key: 'vbus-power',
                    label: 'Set Bus Power State',
                    onClick: openVBUSPowerModal,
                    disabled: !currentDevice.value
                },
                { key: 'sep-3', separator: true },
                { key: 'run-cli', label: 'Run CLI Commands', onClick: runCli },
                { key: 'sep-3', separator: true },
                { key: 'clear-password', label: 'Clear Password', onClick: runCli },
            ]
        },
        {
            key: 'help',
            label: 'Help',
            items: [
                { key: 'documentation', label: 'User Manual', onClick: openDocumentationModal },
                { key: 'sep-1', separator: true },
                { key: 'about', label: 'About', onClick: showAbout }
            ]
        }
    ])
}
