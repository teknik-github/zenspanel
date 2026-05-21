import { ref, shallowRef } from 'vue'

// Toast notifications
export type ToastType = 'success' | 'error' | 'warning' | 'info'

interface Toast {
  id: number
  message: string
  type: ToastType
}

const toasts = ref<Toast[]>([])
let nextId = 0

export function useToast() {
  function show(message: string, type: ToastType = 'info', duration = 4000) {
    const id = ++nextId
    toasts.value.push({ id, message, type })
    setTimeout(() => {
      toasts.value = toasts.value.filter(t => t.id !== id)
    }, duration)
  }

  return {
    toasts,
    success: (msg: string) => show(msg, 'success'),
    error:   (msg: string) => show(msg, 'error', 6000),
    warning: (msg: string) => show(msg, 'warning'),
    info:    (msg: string) => show(msg, 'info'),
  }
}

// Confirm dialog
interface ConfirmOptions {
  title?: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
}

const confirmVisible = ref(false)
const confirmOptions = ref<ConfirmOptions>({ message: '' })
let confirmResolve: ((v: boolean) => void) | null = null

export function useConfirm() {
  function confirm(options: ConfirmOptions | string): Promise<boolean> {
    confirmOptions.value = typeof options === 'string' ? { message: options } : options
    confirmVisible.value = true
    return new Promise(resolve => {
      confirmResolve = resolve
    })
  }

  function resolve(value: boolean) {
    confirmVisible.value = false
    confirmResolve?.(value)
    confirmResolve = null
  }

  return { confirmVisible, confirmOptions, confirm, resolve }
}
