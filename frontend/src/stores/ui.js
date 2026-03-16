import { defineStore } from 'pinia'
import { reactive } from 'vue'

export const useUIStore = defineStore('ui', () => {
    const alert = reactive({
        show: false,
        message: '',
        title: 'Information'
    })

    const showAlert = (message, title = 'Information') => {
        alert.message = message
        alert.title = title
        alert.show = true
    }

    const closeAlert = () => {
        alert.show = false
    }

    return {
        alert,
        showAlert,
        closeAlert
    }
})
