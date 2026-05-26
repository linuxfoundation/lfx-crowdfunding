// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref } from 'vue';
import { authState, login } from '~/composables/useAuth';
import { parseDollarsToCents } from '~/utils/currency';
import type {
  InitiativeType,
  ProjectFormData,
  SecurityAuditFormData,
  GeneralFundFormData,
  EventFormData,
} from '~/types/fundraise.types';

interface FundraiseFormData {
  projectForm?: ProjectFormData | null;
  securityAuditForm?: SecurityAuditFormData | null;
  generalFundForm?: GeneralFundFormData | null;
  eventForm?: EventFormData | null;
}

// Module-level — shared across all callers; only one submission should be in
// flight at a time (the submit button is disabled while submitting.value is true).
const submitting = ref(false);
const error = ref<string | null>(null);

export const useFundraiseSubmit = () => {
  const { showError } = useErrorToast();

  const submitFundraise = async (type: InitiativeType, forms: FundraiseFormData): Promise<void> => {
    if (!authState.value.isAuthenticated) {
      await login(); // redirects to Auth0 with returnTo = current path; does not return
      return;
    }

    submitting.value = true;
    error.value = null;
    try {
      await $fetch('/api/fundraise', {
        method: 'POST',
        body: buildPayload(type, forms),
      });
    } catch (e: unknown) {
      const err = e as { data?: { message?: string }; message?: string };
      const message =
        err?.data?.message ?? err?.message ?? 'Failed to submit initiative. Please try again.';
      error.value = message;
      showError(message);
      throw e;
    } finally {
      submitting.value = false;
    }
  };

  return { submitting, error, submitFundraise };
};

function buildPayload(type: InitiativeType, forms: FundraiseFormData): Record<string, unknown> {
  const { projectForm, securityAuditForm, generalFundForm, eventForm } = forms;

  switch (type) {
    case 'project': {
      const repoUrl = projectForm?.selectedRepo
        ? projectForm.hostingType === 'github'
          ? `https://github.com/${projectForm.selectedRepo}`
          : projectForm.selectedRepo
        : undefined;
      return {
        initiativeType: 'project',
        name: projectForm?.details.projectName ?? '',
        description: projectForm?.details.elevatorPitch ?? '',
        websiteUrl: projectForm?.details.websiteUrl || undefined,
        cocUrl: projectForm?.details.codeOfConductUrl || undefined,
        repositoryUrl: repoUrl,
        beneficiaries: projectForm?.details.beneficiaries?.length
          ? projectForm.details.beneficiaries
          : undefined,
        annualFundingGoalCents: parseDollarsToCents(projectForm?.details.annualFundingGoal),
      };
    }

    case 'security_audit': {
      return {
        initiativeType: 'security_audit',
        name: securityAuditForm?.auditName ?? '',
        description: securityAuditForm?.elevatorPitch ?? '',
        websiteUrl: securityAuditForm?.websiteUrl || undefined,
        repositoryUrl: securityAuditForm?.repositoryUrl || undefined,
        fundingGoalCents: parseDollarsToCents(securityAuditForm?.fundingGoal),
        primaryContact: securityAuditForm?.primaryContact,
        secondaryContact: securityAuditForm?.secondaryContact,
        technicalLead: securityAuditForm?.technicalLead,
      };
    }

    case 'event': {
      return {
        initiativeType: 'event',
        name: eventForm?.name ?? '',
        description: eventForm?.elevatorPitch ?? '',
        websiteUrl: eventForm?.websiteUrl || undefined,
        registrationUrl: eventForm?.registrationUrl || undefined,
        startDate: eventForm?.startDate || undefined,
        endDate: eventForm?.endDate || undefined,
        city: eventForm?.city || undefined,
        country: eventForm?.country || undefined,
        beneficiaries: eventForm?.beneficiaries?.length ? eventForm.beneficiaries : undefined,
        sponsorshipGoalCents: parseDollarsToCents(eventForm?.sponsorshipGoal),
      };
    }

    case 'general_fund': {
      return {
        initiativeType: 'general_fund',
        name: generalFundForm?.name ?? '',
        description: generalFundForm?.elevatorPitch ?? '',
        websiteUrl: generalFundForm?.websiteUrl || undefined,
        beneficiaries: generalFundForm?.beneficiaries?.length
          ? generalFundForm.beneficiaries
          : undefined,
        annualFundingGoalCents: parseDollarsToCents(generalFundForm?.annualFundingGoal),
      };
    }
  }
}
