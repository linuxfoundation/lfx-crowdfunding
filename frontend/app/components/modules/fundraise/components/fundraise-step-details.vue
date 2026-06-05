<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <!-- Project -->
  <fundraise-project-steps
    v-if="initiativeType === 'project'"
    :current-step="subStep"
    :model-value="projectForm"
    @update:model-value="projectForm = $event"
  />

  <!-- Security Audit -->
  <fundraise-security-audit-steps
    v-else-if="initiativeType === 'security_audit'"
    :current-step="subStep"
    :model-value="securityAuditForm"
    @update:model-value="securityAuditForm = $event"
  />

  <!-- General Fund -->
  <fundraise-general-fund-steps
    v-else-if="initiativeType === 'general_fund'"
    :current-step="subStep"
    :model-value="generalFundForm"
    @update:model-value="generalFundForm = $event"
  />

  <!-- Event -->
  <fundraise-event-steps
    v-else-if="initiativeType === 'event'"
    :current-step="subStep"
    :model-value="eventForm"
    @update:model-value="eventForm = $event"
  />
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import FundraiseProjectSteps from './project-steps/fundraise-project-steps.vue';
import FundraiseSecurityAuditSteps from './security-audit-steps/fundraise-security-audit-steps.vue';
import FundraiseGeneralFundSteps from './general-fund-steps/fundraise-general-fund-steps.vue';
import FundraiseEventSteps from './event-steps/fundraise-event-steps.vue';
import type {
  InitiativeType,
  ProjectFormData,
  GoalItem,
  SecurityAuditFormData,
  ContactPerson,
  GeneralFundFormData,
  EventFormData,
} from '~/types/fundraise.types';

const props = defineProps<{
  initiativeType: InitiativeType | null;
}>();

const DEFAULT_FUND_DISTRIBUTION: GoalItem[] = [
  {
    category: 'development',
    label: 'Development',
    description: 'Pay your top developers, and bring in new talent to add features and fix bugs.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'marketing',
    label: 'Marketing',
    description: 'Promote and grow your project through collateral, website redesign, or event swag.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'meetups',
    label: 'Meetups',
    description: 'Connect with your community through local meetups or industry events.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'bug_bounty',
    label: 'Bug Bounty',
    description: 'Have your community help identify bugs and get rewarded.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'travel',
    label: 'Travel',
    description: 'Send project members to conferences, meetups, or customer meetings.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'documentation',
    label: 'Documentation',
    description: 'Drive specific documentation initiatives within your project.',
    enabled: false,
    percentage: 0,
  },
];

const createInitialProjectForm = (): ProjectFormData => ({
  hostingType: null,
  selectedRepo: null,
  details: {
    projectName: '',
    elevatorPitch: '',
    topics: [],
    websiteUrl: '',
    ciiProjectId: '',
    codeOfConductUrl: '',
    logoUrl: '',
    beneficiaries: [],
    annualFundingGoal: '',
    goals: DEFAULT_FUND_DISTRIBUTION.map((item) => ({ ...item })),
  },
  compliance: { ofacConfirmed: false, termsAccepted: false },
});

const createInitialContact = (): ContactPerson => ({
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  preferredContact: 'email',
});

const createInitialSecurityAuditForm = (): SecurityAuditFormData => ({
  auditName: '',
  elevatorPitch: '',
  topics: [],
  repositoryUrl: '',
  websiteUrl: '',
  logoUrl: '',
  ciiProjectId: '',
  licenseType: '',
  currentSecurityStrategy: '',
  codeOfConductUrl: '',
  primaryContact: createInitialContact(),
  secondaryContact: createInitialContact(),
  technicalLead: createInitialContact(),
  fundingGoal: '',
  compliance: { ofacConfirmed: false, termsAccepted: false },
});

const DEFAULT_BUDGET_DISTRIBUTION: GoalItem[] = [
  {
    category: 'venue',
    label: 'Venue',
    description: 'Make sure you have the funds for the right venue to accommodate your event.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'food_beverage',
    label: 'Food & Beverage',
    description: 'Provide coffee, snacks, or cover meals during your event.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'travel',
    label: 'Travel',
    description: 'Provide travel assistance to community members to attend your event.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'equipment',
    label: 'Equipment',
    description: 'Budget for microphones, projectors, screens, and other equipment.',
    enabled: false,
    percentage: 0,
  },
  {
    category: 'marketing',
    label: 'Marketing',
    description: 'Promote your event with advertising, attendee swag, or website design.',
    enabled: false,
    percentage: 0,
  },
];

const createInitialEventForm = (): EventFormData => ({
  name: '',
  elevatorPitch: '',
  topics: [],
  websiteUrl: '',
  registrationUrl: '',
  startDate: '',
  endDate: '',
  city: '',
  country: '',
  logoUrl: '',
  beneficiaries: [],
  sponsorshipGoal: '',
  budgetDistribution: DEFAULT_BUDGET_DISTRIBUTION.map((item) => ({ ...item })),
  compliance: { ofacConfirmed: false, termsAccepted: false },
});

const createInitialGeneralFundForm = (): GeneralFundFormData => ({
  name: '',
  elevatorPitch: '',
  topics: [],
  websiteUrl: '',
  logoUrl: '',
  beneficiaries: [],
  annualFundingGoal: '',
  compliance: { ofacConfirmed: false, termsAccepted: false },
});

const subStep = ref(0);
const projectForm = ref<ProjectFormData>(createInitialProjectForm());
const securityAuditForm = ref<SecurityAuditFormData>(createInitialSecurityAuditForm());
const generalFundForm = ref<GeneralFundFormData>(createInitialGeneralFundForm());
const eventForm = ref<EventFormData>(createInitialEventForm());

const totalSubSteps = computed(() => {
  if (props.initiativeType === 'project') {
    return projectForm.value.hostingType === 'github' ? 4 : 3;
  }
  if (props.initiativeType === 'security_audit') return 2;
  if (props.initiativeType === 'general_fund') return 2;
  if (props.initiativeType === 'event') return 2;
  return 1;
});

const isLastSubStep = computed(() => subStep.value === totalSubSteps.value - 1);

const isCurrentSubStepValid = computed(() => {
  if (props.initiativeType === 'project') {
    if (subStep.value === 0) return projectForm.value.hostingType !== null;
    if (subStep.value === 1 && projectForm.value.hostingType === 'github') {
      return projectForm.value.selectedRepo !== null;
    }
    if (subStep.value === totalSubSteps.value - 1) {
      return projectForm.value.compliance.ofacConfirmed && projectForm.value.compliance.termsAccepted;
    }
    return true;
  }
  if (props.initiativeType === 'security_audit') {
    if (subStep.value === totalSubSteps.value - 1) {
      return securityAuditForm.value.compliance.ofacConfirmed && securityAuditForm.value.compliance.termsAccepted;
    }
    return securityAuditForm.value.auditName.trim() !== '';
  }
  if (props.initiativeType === 'general_fund') {
    if (subStep.value === totalSubSteps.value - 1) {
      return generalFundForm.value.compliance.ofacConfirmed && generalFundForm.value.compliance.termsAccepted;
    }
    return generalFundForm.value.name.trim() !== '';
  }
  if (props.initiativeType === 'event') {
    if (subStep.value === totalSubSteps.value - 1) {
      return eventForm.value.compliance.ofacConfirmed && eventForm.value.compliance.termsAccepted;
    }
    return eventForm.value.name.trim() !== '';
  }
  return true;
});

watch(
  () => projectForm.value.selectedRepo,
  (repo) => {
    if (repo && !projectForm.value.details.projectName) {
      projectForm.value.details.projectName = repo.split('/').pop() ?? repo;
    }
  },
);

const advance = () => {
  if (!isLastSubStep.value) subStep.value++;
};

const back = () => {
  if (subStep.value > 0) subStep.value--;
};

const reset = () => {
  subStep.value = 0;
  projectForm.value = createInitialProjectForm();
  securityAuditForm.value = createInitialSecurityAuditForm();
  generalFundForm.value = createInitialGeneralFundForm();
  eventForm.value = createInitialEventForm();
};

defineExpose({
  subStep,
  totalSubSteps,
  isLastSubStep,
  isCurrentSubStepValid,
  advance,
  back,
  reset,
  projectForm,
  securityAuditForm,
  generalFundForm,
  eventForm,
});
</script>

<script lang="ts">
export default {
  name: 'FundraiseStepDetails',
};
</script>
