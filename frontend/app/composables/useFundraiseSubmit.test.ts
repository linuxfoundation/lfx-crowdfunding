// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import { ref } from 'vue';

// useAuth exports authState (a ref) and login directly — mock the module so they're controllable.
// vi.hoisted can't use `ref` (it's hoisted above imports), so we create the ref after import
// and replace the mock's return value in beforeEach.
const mockLogin = vi.fn();
const mockAuthState = ref({ isAuthenticated: false, user: null as null, token: null as null });

vi.mock('~/composables/useAuth', () => ({
  get authState() {
    return mockAuthState;
  },
  login: (...args: unknown[]) => mockLogin(...args),
}));

// useErrorToast is a Nuxt auto-import — stub on globalThis.
const mockShowError = vi.fn();
// @ts-expect-error — Nuxt auto-import stub
globalThis.useErrorToast = () => ({ showError: mockShowError });

const mockFetch = vi.fn();
globalThis.$fetch = mockFetch as typeof $fetch;

import { useFundraiseSubmit } from './useFundraiseSubmit';

// Minimal valid form data per initiative type.
const projectForms = {
  projectForm: {
    hostingType: 'github' as const,
    selectedRepo: null,
    details: {
      projectName: 'Test Project',
      elevatorPitch: 'A test project',
      topics: [],
      websiteUrl: '',
      ciiProjectId: '',
      codeOfConductUrl: '',
      logoUrl: '',
      beneficiaries: [],
      annualFundingGoal: '',
      goals: [],
    },
  },
};

describe('useFundraiseSubmit', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthState.value = { isAuthenticated: false, user: null, token: null };
    const { submitting, error } = useFundraiseSubmit();
    submitting.value = false;
    error.value = null;
  });

  describe('submitFundraise — unauthenticated user', () => {
    it('calls login() and does not call $fetch when not authenticated', async () => {
      mockAuthState.value.isAuthenticated = false;

      const { submitFundraise } = useFundraiseSubmit();
      await submitFundraise('project', projectForms);

      expect(mockLogin).toHaveBeenCalledOnce();
      expect(mockFetch).not.toHaveBeenCalled();
    });
  });

  describe('submitFundraise — authenticated user', () => {
    beforeEach(() => {
      mockAuthState.value.isAuthenticated = true;
    });

    it('calls POST /api/fundraise', async () => {
      mockFetch.mockResolvedValueOnce(undefined);

      const { submitFundraise } = useFundraiseSubmit();
      await submitFundraise('project', projectForms);

      expect(mockFetch).toHaveBeenCalledOnce();
      const [url, opts] = mockFetch.mock.calls[0] as [string, { method: string; body: unknown }];
      expect(url).toBe('/api/fundraise');
      expect(opts.method).toBe('POST');
    });

    it('sets submitting=true during the call and resets it afterwards', async () => {
      const { submitFundraise, submitting } = useFundraiseSubmit();

      let submittingDuringCall = false;
      mockFetch.mockImplementationOnce(async () => {
        submittingDuringCall = submitting.value;
      });

      await submitFundraise('project', projectForms);

      expect(submittingDuringCall).toBe(true);
      expect(submitting.value).toBe(false);
    });

    it('resets submitting=false even when the call throws', async () => {
      mockFetch.mockRejectedValueOnce(new Error('fail'));

      const { submitFundraise, submitting } = useFundraiseSubmit();
      await expect(submitFundraise('project', projectForms)).rejects.toBeTruthy();
      expect(submitting.value).toBe(false);
    });
  });

  describe('submitFundraise — error handling', () => {
    beforeEach(() => {
      mockAuthState.value.isAuthenticated = true;
    });

    it('sets error and calls showError on $fetch failure', async () => {
      mockFetch.mockRejectedValueOnce({ data: { message: 'Validation failed' } });

      const { submitFundraise, error } = useFundraiseSubmit();
      await expect(submitFundraise('project', projectForms)).rejects.toBeTruthy();
      expect(error.value).toBe('Validation failed');
      expect(mockShowError).toHaveBeenCalledWith('Validation failed');
    });

    it('uses message fallback when error has no data.message', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const { submitFundraise, error } = useFundraiseSubmit();
      await expect(submitFundraise('project', projectForms)).rejects.toBeTruthy();
      expect(error.value).toBe('Network error');
    });

    it('uses default message when error has no message at all', async () => {
      mockFetch.mockRejectedValueOnce({});

      const { submitFundraise, error } = useFundraiseSubmit();
      await expect(submitFundraise('project', projectForms)).rejects.toBeTruthy();
      expect(error.value).toBe('Failed to submit initiative. Please try again.');
    });

    it('re-throws the original error so callers can handle it', async () => {
      const originalError = { data: { message: 'Validation failed' }, statusCode: 422 };
      mockFetch.mockRejectedValueOnce(originalError);

      const { submitFundraise } = useFundraiseSubmit();
      await expect(submitFundraise('project', projectForms)).rejects.toMatchObject({
        statusCode: 422,
      });
    });
  });

  describe('buildPayload — per initiative type', () => {
    beforeEach(() => {
      mockAuthState.value.isAuthenticated = true;
      mockFetch.mockResolvedValue(undefined);
    });

    it('sends initiativeType=project with projectForm data', async () => {
      const { submitFundraise } = useFundraiseSubmit();
      await submitFundraise('project', {
        projectForm: {
          hostingType: 'github',
          selectedRepo: 'org/repo',
          details: {
            projectName: 'My Project',
            elevatorPitch: 'Great project',
            topics: ['cloud', 'security'],
            websiteUrl: 'https://example.com',
            ciiProjectId: '',
            codeOfConductUrl: '',
            logoUrl: 'https://logo.example.com',
            beneficiaries: ['org1'],
            annualFundingGoal: '50000',
            goals: ['goal1'],
          },
        },
      });

      const body = (mockFetch.mock.calls[0] as [string, { body: Record<string, unknown> }])[1].body;
      expect(body.initiativeType).toBe('project');
      expect(body.name).toBe('My Project');
      expect(body.description).toBe('Great project');
      expect(body.industry).toBe('cloud,security');
      expect(body.repositoryUrl).toBe('https://github.com/org/repo');
      expect(body.annualFundingGoalCents).toBeGreaterThan(0);
    });

    it('sends initiativeType=security_audit with securityAuditForm data', async () => {
      const { submitFundraise } = useFundraiseSubmit();
      await submitFundraise('security_audit', {
        securityAuditForm: {
          auditName: 'Audit 2024',
          elevatorPitch: 'Security audit for X',
          topics: ['security'],
          websiteUrl: '',
          ciiProjectId: '',
          codeOfConductUrl: '',
          repositoryUrl: '',
          logoUrl: '',
          licenseType: 'MIT',
          currentSecurityStrategy: 'none',
          fundingGoal: '10000',
          primaryContact: { name: 'Alice', email: 'alice@example.com' },
          secondaryContact: undefined,
          technicalLead: undefined,
        },
      });

      const body = (mockFetch.mock.calls[0] as [string, { body: Record<string, unknown> }])[1].body;
      expect(body.initiativeType).toBe('security_audit');
      expect(body.name).toBe('Audit 2024');
      expect(body.licenseType).toBe('MIT');
    });

    it('sends initiativeType=event with eventForm data', async () => {
      const { submitFundraise } = useFundraiseSubmit();
      await submitFundraise('event', {
        eventForm: {
          name: 'LFX Summit',
          elevatorPitch: 'Annual summit',
          topics: [],
          websiteUrl: '',
          registrationUrl: '',
          startDate: '2024-06-01',
          endDate: '2024-06-03',
          city: 'San Francisco',
          country: 'US',
          logoUrl: '',
          beneficiaries: [],
          sponsorshipGoal: '25000',
          budgetDistribution: [],
        },
      });

      const body = (mockFetch.mock.calls[0] as [string, { body: Record<string, unknown> }])[1].body;
      expect(body.initiativeType).toBe('event');
      expect(body.name).toBe('LFX Summit');
      expect(body.city).toBe('San Francisco');
      expect(body.country).toBe('US');
    });

    it('sends initiativeType=general_fund with generalFundForm data', async () => {
      const { submitFundraise } = useFundraiseSubmit();
      await submitFundraise('general_fund', {
        generalFundForm: {
          name: 'General Fund',
          elevatorPitch: 'A general fund',
          topics: [],
          websiteUrl: '',
          logoUrl: '',
          beneficiaries: [],
          annualFundingGoal: '100000',
        },
      });

      const body = (mockFetch.mock.calls[0] as [string, { body: Record<string, unknown> }])[1].body;
      expect(body.initiativeType).toBe('general_fund');
      expect(body.name).toBe('General Fund');
      expect(body.annualFundingGoalCents).toBeGreaterThan(0);
    });
  });
});
