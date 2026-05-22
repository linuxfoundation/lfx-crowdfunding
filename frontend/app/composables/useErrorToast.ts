// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import useToastService from '~/components/uikit/toast/toast.service';
import { ToastTypesEnum } from '~/components/uikit/toast/types/toast.types';

export const useErrorToast = () => {
  const { showToast } = useToastService();

  const showError = (message: string) => {
    showToast(message, ToastTypesEnum.negative);
  };

  return { showError };
};
