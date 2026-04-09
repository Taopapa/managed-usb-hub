import { ref } from 'vue'

export const useMenuModals = ({ currentDevice, showAlert, closeMenu }) => {
    const showDocModal = ref(false)
    const showVBUSPowerModal = ref(false)
    const showDeviceNameModal = ref(false)

    const ensureDeviceSelected = () => {
        if (currentDevice.value) {
            return true
        }

        showAlert('Please connect a device first.', 'Connection Required')
        closeMenu()
        return false
    }

    const openDeviceNameModal = () => {
        if (!ensureDeviceSelected()) return

        showDeviceNameModal.value = true
        closeMenu()
    }

    const closeDeviceNameModal = () => {
        showDeviceNameModal.value = false
    }

    const openVBUSPowerModal = () => {
        if (!ensureDeviceSelected()) return

        showVBUSPowerModal.value = true
        closeMenu()
    }

    const closeVBUSPowerModal = () => {
        showVBUSPowerModal.value = false
    }

    const openDocumentationModal = () => {
        showDocModal.value = true
        closeMenu()
    }

    const closeDocumentationModal = () => {
        showDocModal.value = false
    }

    return {
        showDocModal,
        showVBUSPowerModal,
        showDeviceNameModal,
        openDeviceNameModal,
        closeDeviceNameModal,
        openVBUSPowerModal,
        closeVBUSPowerModal,
        openDocumentationModal,
        closeDocumentationModal
    }
}
