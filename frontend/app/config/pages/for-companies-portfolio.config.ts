// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface PortfolioInitiative {
  name: string;
}

export interface Portfolio {
  id: string;
  icon: string;
  name: string;
  description: string;
  initiatives: PortfolioInitiative[];
  suggestedFunding: string;
}

export const portfolios: Portfolio[] = [
  {
    id: 'security',
    icon: 'shield-check',
    name: 'Security Portfolio',
    description:
      'The Linux Foundation is a 501(c)(6) nonprofit. Contributions may be tax-deductible as business expenses. We provide full documentation for your finance team.',
    initiatives: [
      { name: 'Kubernetes Security Audit' },
      { name: 'OpenSSF Scorecard' },
      { name: 'CPython Security Audit' },
      { name: 'Zephyr RTOS Security' },
    ],
    suggestedFunding: '$100,000+',
  },
  {
    id: 'infrastructure',
    icon: 'layer-group',
    name: 'Infrastructure Bundle',
    description:
      'Sustain the infrastructure projects your engineering teams rely on daily. From TLS certificates to package managers.',
    initiatives: [
      { name: "Let's Encrypt Infrastructure" },
      { name: 'Node.js Sustainability' },
      { name: 'curl Maintenance Fund' },
    ],
    suggestedFunding: '$75,000+',
  },
  {
    id: 'talent',
    icon: 'people-group',
    name: 'Talent Pipeline',
    description:
      'Invest in the next generation of open source contributors. Fund mentorships, travel grants, and education programs.',
    initiatives: [
      { name: 'Linux Kernel Mentorship' },
      { name: 'Kernel Developer Travel Fund' },
      { name: 'Hyperledger Education' },
    ],
    suggestedFunding: '$50,000+',
  },
];
