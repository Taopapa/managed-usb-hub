import { computed } from 'vue'

export const useAppMenus = ({
    currentDevice,
    autoSearch,
    exportLogs,
    quitApp,
    refreshHub,
    openDeviceNameModal,
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
                    label: 'Device Name',
                    onClick: openDeviceNameModal,
                    disabled: !currentDevice.value
                },
                { key: 'sep-1', separator: true },
                {
                    key: 'vbus-power',
                    label: 'Vbus Power Option',
                    onClick: openVBUSPowerModal,
                    disabled: !currentDevice.value
                },
                { key: 'sep-2', separator: true },
                { key: 'run-cli', label: 'Run CLI Command', onClick: runCli }
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
