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

    const showConfirm = (message, title = 'Confirm') => {
        return new Promise((resolve) => {
            confirmState.message = message
            confirmState.title = title
            confirmState.show = true
            confirmState.resolve = resolve
        })
    }

    const handleConfirmResult = (result) => {
        confirmState.show = false
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
