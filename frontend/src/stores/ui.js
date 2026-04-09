import { defineStore } from 'pinia'
import { reactive } from 'vue'

export const useUIStore = defineStore('ui', () => {
    const alert = reactive({
        show: false,
        message: '',
        title: 'Information'
    })

    const confirmState = reactive({
        show: false,
        message: '',
        title: 'Confirm',
        confirmLabel: 'OK',
        cancelLabel: 'Cancel',
        resolve: null
    })

    const showAlert = (message, title = 'Information') => {
        alert.message = message
        alert.title = title
        alert.show = true
    }

    const closeAlert = () => {
        alert.show = false
    }

    const showConfirm = (message, title = 'Confirm', options = {}) => {
        return new Promise((resolve) => {
            confirmState.message = message
            confirmState.title = title
            confirmState.confirmLabel = options.confirmLabel || 'OK'
            confirmState.cancelLabel = options.cancelLabel || 'Cancel'
            confirmState.show = true
            confirmState.resolve = resolve
        })
    }

    const handleConfirmResult = (result) => {
        confirmState.show = false
        confirmState.confirmLabel = 'OK'
        confirmState.cancelLabel = 'Cancel'
        if (confirmState.resolve) {
            confirmState.resolve(result)
            confirmState.resolve = null
        }
    }

    return {
        alert,
        showAlert,
        closeAlert,
        confirmState,
        showConfirm,
        handleConfirmResult
    }
})
