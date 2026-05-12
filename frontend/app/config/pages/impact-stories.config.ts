// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface ImpactStoryTag {
  label: string;
  value: string;
}

export interface ImpactStory {
  id: string;
  initiativeName: string;
  logoSrc?: string;
  tags: ImpactStoryTag[];
  quote: string;
  attribution: string;
}

export const impactStories: ImpactStory[] = [
  {
    id: 'kubernetes-security-audit',
    initiativeName: 'Kubernetes Security Audit',
    logoSrc:
      'https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.svg',
    tags: [
      { label: 'Vulnerabilities found', value: '23' },
      { label: 'CVEs prevented', value: '7' },
      { label: 'lines Code paths reviewed', value: '1.2M' },
    ],
    quote:
      '"This audit found vulnerabilities we never would have caught internally. It\'s the most important security investment in Kubernetes this year."',
    attribution: 'Tim Allclair, Kubernetes Security Lead',
  },
  {
    id: 'linux-kernel-mentorship',
    initiativeName: 'Linux Kernel Mentorship',
    logoSrc: 'https://www.linuxfoundation.org/hubfs/lf-stacked-color.svg',
    tags: [
      { label: 'Mentees', value: '142' },
      { label: 'First-time contributors', value: '38' },
      { label: 'Subsystem maintainers', value: '12' },
    ],
    quote:
      '"The mentorship program changed my career. I went from zero kernel knowledge to maintaining a subsystem in 18 months."',
    attribution: 'Aisha Patel, Linux Kernel Subsystem Maintainer',
  },
  {
    id: 'lets-encrypt-infrastructure',
    initiativeName: "Let's Encrypt Infrastructure",
    logoSrc: 'https://letsencrypt.org/images/le-logo-standard.svg',
    tags: [
      { label: 'Certificates', value: '450M+' },
      { label: 'Uptime', value: '99.99%' },
      { label: 'Free for everyone', value: '100%' },
    ],
    quote:
      '"Let\'s Encrypt made HTTPS the default for the entire web. This fund keeps it running for everyone, for free."',
    attribution: 'Josh Aas, ISRG Executive Director',
  },
];
