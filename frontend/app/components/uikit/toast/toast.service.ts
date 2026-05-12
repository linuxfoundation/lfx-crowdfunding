// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { useToast } from 'primevue/usetoast';
import type { ToastOptions, ToastSeverity, ToastType } from './types/toast.types';

const useToastService = () => {
  const toast = useToast();
  const showToast = (
    message: string,
    toastType: ToastType,
    icon?: string,
    delay: number = 3000,
    config: Partial<ToastOptions> = {},
  ) => {
    const toastOptions: ToastOptions = {
      severity: toastType as ToastSeverity,
      summary: toastType,
      detail: message,
      closable: false,
      icon,
      life: delay,
      ...config,
    };
    toast.add(toastOptions);
  };

  return {
    showToast,
  };
};

export default useToastService;
