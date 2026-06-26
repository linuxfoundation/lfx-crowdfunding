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
      const repoUrl =
        projectForm?.hostingType === 'github'
          ? projectForm.selectedRepo
            ? `https://github.com/${projectForm.selectedRepo}`
            : undefined
          : projectForm?.details.repositoryUrl || undefined;
      return {
        initiativeType: 'project',
        name: projectForm?.details.projectName ?? '',
        description: projectForm?.details.elevatorPitch ?? '',
        industry: projectForm?.details.topics?.length
          ? projectForm.details.topics.join(',')
          : undefined,
        websiteUrl: projectForm?.details.websiteUrl || undefined,
        ciiProjectId: projectForm?.details.ciiProjectId || undefined,
        cocUrl: projectForm?.details.codeOfConductUrl || undefined,
        repositoryUrl: repoUrl,
        logoUrl: projectForm?.details.logoUrl || undefined,
        beneficiaries: projectForm?.details.beneficiaries?.length
          ? projectForm.details.beneficiaries
          : undefined,
        annualFundingGoalCents: parseDollarsToCents(projectForm?.details.annualFundingGoal),
        goals: projectForm?.details.goals?.length ? projectForm.details.goals : undefined,
      };
    }

    case 'security_audit': {
      return {
        initiativeType: 'security_audit',
        name: securityAuditForm?.auditName ?? '',
        description: securityAuditForm?.elevatorPitch ?? '',
        industry: securityAuditForm?.topics?.length
          ? securityAuditForm.topics.join(',')
          : undefined,
        websiteUrl: securityAuditForm?.websiteUrl || undefined,
        ciiProjectId: securityAuditForm?.ciiProjectId || undefined,
        cocUrl: securityAuditForm?.codeOfConductUrl || undefined,
        repositoryUrl: securityAuditForm?.repositoryUrl || undefined,
        logoUrl: securityAuditForm?.logoUrl || undefined,
        licenseType: securityAuditForm?.licenseType || undefined,
        currentSecurityStrategy: securityAuditForm?.currentSecurityStrategy || undefined,
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
        industry: eventForm?.topics?.length ? eventForm.topics.join(',') : undefined,
        websiteUrl: eventForm?.websiteUrl || undefined,
        registrationUrl: eventForm?.registrationUrl || undefined,
        startDate: eventForm?.startDate || undefined,
        endDate: eventForm?.endDate || undefined,
        city: eventForm?.city || undefined,
        country: eventForm?.country || undefined,
        logoUrl: eventForm?.logoUrl || undefined,
        beneficiaries: eventForm?.beneficiaries?.length ? eventForm.beneficiaries : undefined,
        sponsorshipGoalCents: parseDollarsToCents(eventForm?.sponsorshipGoal),
        budgetDistribution: eventForm?.budgetDistribution?.length
          ? eventForm.budgetDistribution
          : undefined,
      };
    }

    case 'general_fund': {
      return {
        initiativeType: 'general_fund',
        name: generalFundForm?.name ?? '',
        description: generalFundForm?.elevatorPitch ?? '',
        industry: generalFundForm?.topics?.length ? generalFundForm.topics.join(',') : undefined,
        websiteUrl: generalFundForm?.websiteUrl || undefined,
        logoUrl: generalFundForm?.logoUrl || undefined,
        beneficiaries: generalFundForm?.beneficiaries?.length
          ? generalFundForm.beneficiaries
          : undefined,
        annualFundingGoalCents: parseDollarsToCents(generalFundForm?.annualFundingGoal),
      };
    }
  }
}
